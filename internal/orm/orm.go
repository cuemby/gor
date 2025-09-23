package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/cuemby/gor/pkg/gor"
)

// gorORM implements the gor.ORM interface
type gorORM struct {
	db       *sql.DB
	adapter  gor.DatabaseAdapter
	models   map[string]reflect.Type
	migrator *Migrator
	config   gor.DatabaseConfig
}

// NewORM creates a new ORM instance
func NewORM(config gor.DatabaseConfig) gor.ORM {
	return &gorORM{
		models:  make(map[string]reflect.Type),
		config:  config,
		adapter: getAdapter(config.Driver),
	}
}

// Connect establishes database connection
func (o *gorORM) Connect(ctx context.Context, config gor.DatabaseConfig) error {
	db, err := o.adapter.Connect(config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	o.db = db
	o.migrator = NewMigrator(db, o.adapter)

	// Configure connection pool
	o.db.SetMaxOpenConns(config.MaxOpenConns)
	o.db.SetMaxIdleConns(config.MaxIdleConns)
	o.db.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Test connection
	return o.db.PingContext(ctx)
}

// Close closes the database connection
func (o *gorORM) Close() error {
	if o.db != nil {
		return o.db.Close()
	}
	return nil
}

// DB returns the underlying sql.DB
func (o *gorORM) DB() *sql.DB {
	return o.db
}

// Register registers models with the ORM
func (o *gorORM) Register(models ...interface{}) error {
	for _, model := range models {
		t := reflect.TypeOf(model)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}

		// Get table name
		tableName := getTableName(t)
		o.models[tableName] = t

		// Create table if auto-migrate is enabled
		if err := o.createTableIfNotExists(t); err != nil {
			return fmt.Errorf("failed to create table for %s: %w", tableName, err)
		}
	}
	return nil
}

// Table returns a table instance
func (o *gorORM) Table(name string) gor.Table {
	if modelType, exists := o.models[name]; exists {
		return NewTable(name, modelType, o.db, o.adapter)
	}
	return NewTable(name, nil, o.db, o.adapter)
}

// Transaction executes a function within a database transaction
func (o *gorORM) Transaction(ctx context.Context, fn func(tx gor.Transaction) error) error {
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	gorTx := &gorTransaction{tx: tx, orm: o}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(gorTx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx error: %v, rollback error: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// Query creates a new query builder
func (o *gorORM) Query(model interface{}) gor.QueryBuilder {
	return NewQueryBuilder(model, o.db, o.adapter)
}

// Find finds a record by ID
func (o *gorORM) Find(model interface{}, id interface{}) error {
	qb := o.Query(model)
	return qb.Where("id = ?", id).First(model)
}

// FindAll finds all records
func (o *gorORM) FindAll(models interface{}) error {
	qb := o.Query(models)
	return qb.FindAll(models)
}

// Create creates a new record
func (o *gorORM) Create(model interface{}) error {
	// Call BeforeCreate hook if model implements it
	if hook, ok := model.(interface{ BeforeCreate() error }); ok {
		if err := hook.BeforeCreate(); err != nil {
			return err
		}
	}

	// Set timestamps
	setTimestamps(model, true)

	// Generate and execute insert SQL
	sql, args, err := o.adapter.GenerateSQL(NewQueryBuilder(model, o.db, o.adapter))
	if err != nil {
		return err
	}

	result, err := o.db.Exec(sql, args...)
	if err != nil {
		return err
	}

	// Set ID if auto-increment
	if id, err := result.LastInsertId(); err == nil {
		setID(model, id)
	}

	// Call AfterCreate hook if model implements it
	if hook, ok := model.(interface{ AfterCreate() error }); ok {
		if err := hook.AfterCreate(); err != nil {
			return err
		}
	}

	return nil
}

// Update updates an existing record
func (o *gorORM) Update(model interface{}) error {
	// Call BeforeUpdate hook
	if hook, ok := model.(interface{ BeforeUpdate() error }); ok {
		if err := hook.BeforeUpdate(); err != nil {
			return err
		}
	}

	// Set updated timestamp
	setTimestamps(model, false)

	// Generate and execute update SQL
	id := getID(model)
	qb := NewQueryBuilder(model, o.db, o.adapter).Where("id = ?", id)
	sql, args, err := o.adapter.GenerateSQL(qb)
	if err != nil {
		return err
	}

	_, err = o.db.Exec(sql, args...)
	if err != nil {
		return err
	}

	// Call AfterUpdate hook
	if hook, ok := model.(interface{ AfterUpdate() error }); ok {
		if err := hook.AfterUpdate(); err != nil {
			return err
		}
	}

	return nil
}

// Delete deletes a record
func (o *gorORM) Delete(model interface{}) error {
	// Call BeforeDelete hook
	if hook, ok := model.(interface{ BeforeDelete() error }); ok {
		if err := hook.BeforeDelete(); err != nil {
			return err
		}
	}

	id := getID(model)
	tableName := getTableName(reflect.TypeOf(model))

	sql := fmt.Sprintf("DELETE FROM %s WHERE id = ?", tableName)
	_, err := o.db.Exec(sql, id)
	if err != nil {
		return err
	}

	// Call AfterDelete hook
	if hook, ok := model.(interface{ AfterDelete() error }); ok {
		if err := hook.AfterDelete(); err != nil {
			return err
		}
	}

	return nil
}

// Migration methods
func (o *gorORM) Migrate(ctx context.Context) error {
	return o.migrator.Migrate(ctx)
}

func (o *gorORM) Rollback(ctx context.Context, steps int) error {
	return o.migrator.Rollback(ctx, steps)
}

func (o *gorORM) MigrationStatus(ctx context.Context) ([]gor.Migration, error) {
	return o.migrator.Status(ctx)
}

// Helper functions
func getAdapter(driver string) gor.DatabaseAdapter {
	switch driver {
	case "sqlite3", "sqlite":
		return NewSQLiteAdapter()
	case "postgres", "postgresql":
		return NewPostgreSQLAdapter()
	case "mysql":
		return NewMySQLAdapter()
	default:
		return NewSQLiteAdapter() // Default to SQLite
	}
}

func getTableName(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Check if type implements TableName method
	model := reflect.New(t).Interface()
	if tn, ok := model.(interface{ TableName() string }); ok {
		return tn.TableName()
	}

	// Default to lowercase struct name + "s"
	return toSnakeCase(t.Name()) + "s"
}

func setTimestamps(model interface{}, isCreate bool) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	now := time.Now()

	// Set CreatedAt for new records
	if isCreate {
		if field := v.FieldByName("CreatedAt"); field.IsValid() && field.CanSet() {
			field.Set(reflect.ValueOf(now))
		}
	}

	// Always set UpdatedAt
	if field := v.FieldByName("UpdatedAt"); field.IsValid() && field.CanSet() {
		field.Set(reflect.ValueOf(now))
	}
}

func setID(model interface{}, id int64) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if field := v.FieldByName("ID"); field.IsValid() && field.CanSet() {
		switch field.Kind() {
		case reflect.Int, reflect.Int32, reflect.Int64:
			field.SetInt(id)
		case reflect.Uint, reflect.Uint32, reflect.Uint64:
			field.SetUint(uint64(id))
		}
	}
}

func getID(model interface{}) interface{} {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if field := v.FieldByName("ID"); field.IsValid() {
		return field.Interface()
	}
	return nil
}

func (o *gorORM) createTableIfNotExists(t reflect.Type) error {
	tableName := getTableName(t)
	columns := extractColumns(t)

	// Check if table exists
	exists, err := o.tableExists(tableName)
	if err != nil {
		return fmt.Errorf("failed to check if table exists: %w", err)
	}

	if !exists {
		// Create table
		sql := o.adapter.CreateTableSQL(tableName, columns)
		fmt.Printf("Creating table %s with SQL: %s\n", tableName, sql)
		_, err = o.db.Exec(sql)
		if err != nil {
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
		fmt.Printf("✅ Created table: %s\n", tableName)
	} else {
		fmt.Printf("ℹ️  Table %s already exists\n", tableName)
	}

	return nil
}

func (o *gorORM) tableExists(tableName string) (bool, error) {
	var exists bool
	var query string

	if o.config.Driver == "sqlite" || o.config.Driver == "sqlite3" {
		query = "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?"
		var count int
		err := o.db.QueryRow(query, tableName).Scan(&count)
		return count > 0, err
	} else {
		query = "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = ?)"
		err := o.db.QueryRow(query, tableName).Scan(&exists)
		return exists, err
	}
}

func extractColumns(t reflect.Type) []gor.Column {
	var columns []gor.Column

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip embedded structs (like BaseModel)
		if field.Anonymous {
			// Recursively extract fields from embedded struct
			embeddedColumns := extractColumns(field.Type)
			columns = append(columns, embeddedColumns...)
			continue
		}

		column := gor.Column{
			Name: toSnakeCase(field.Name),
			Type: getColumnType(field.Type),
		}

		// Parse struct tags
		if tag := field.Tag.Get("gor"); tag != "" {
			parseStructTag(tag, &column)
		}

		columns = append(columns, column)
	}

	return columns
}

func getColumnType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int, reflect.Int32:
		return "INTEGER"
	case reflect.Int64:
		return "BIGINT"
	case reflect.Uint, reflect.Uint32:
		return "INTEGER"
	case reflect.Uint64:
		return "BIGINT"
	case reflect.String:
		return "TEXT"
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.Float32:
		return "REAL"
	case reflect.Float64:
		return "DOUBLE"
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return "BLOB"
		}
		return "TEXT" // JSON
	default:
		if t == reflect.TypeOf(time.Time{}) {
			return "TIMESTAMP"
		}
		return "TEXT" // JSON fallback
	}
}

func parseStructTag(tag string, column *gor.Column) {
	// Simple tag parsing - can be enhanced
	if contains(tag, "primary_key") {
		column.PrimaryKey = true
	}
	if contains(tag, "not_null") {
		column.Nullable = false
	}
	if contains(tag, "unique") {
		column.Unique = true
	}
	if contains(tag, "index") {
		column.Index = true
	}
	if contains(tag, "auto_increment") {
		// Handle auto increment
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)+1] == substr+";" ||
		s[len(s)-len(substr)-1:] == ";"+substr)))
}

func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	// Special cases for common field names
	switch s {
	case "ID":
		return "id"
	case "UserID":
		return "user_id"
	case "CreatedAt":
		return "created_at"
	case "UpdatedAt":
		return "updated_at"
	}

	// Convert to snake_case
	var result []rune

	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r|0x20) // Convert to lowercase
	}

	return string(result)
}