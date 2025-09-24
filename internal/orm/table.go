package orm

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/cuemby/gor/pkg/gor"
)

// gorTable implements the gor.Table interface
type gorTable struct {
	name      string
	modelType reflect.Type
	db        *sql.DB
	adapter   gor.DatabaseAdapter
	columns   []gor.Column
	indexes   []gor.Index
}

// NewTable creates a new table instance
func NewTable(name string, modelType reflect.Type, db *sql.DB, adapter gor.DatabaseAdapter) gor.Table {
	table := &gorTable{
		name:      name,
		modelType: modelType,
		db:        db,
		adapter:   adapter,
	}

	if modelType != nil {
		table.columns = extractColumns(modelType)
		table.indexes = extractIndexes(modelType, name)
	}

	return table
}

// Name returns the table name
func (t *gorTable) Name() string {
	return t.name
}

// Columns returns the table columns
func (t *gorTable) Columns() []gor.Column {
	return t.columns
}

// Indexes returns the table indexes
func (t *gorTable) Indexes() []gor.Index {
	return t.indexes
}

// Create creates a new record in the table
func (t *gorTable) Create(model interface{}) error {
	// Validate model type
	if err := t.validateModel(model); err != nil {
		return err
	}

	// Call BeforeCreate hook if model implements it
	if hook, ok := model.(interface{ BeforeCreate() error }); ok {
		if err := hook.BeforeCreate(); err != nil {
			return err
		}
	}

	// Set timestamps
	setTimestamps(model, true)

	// Generate insert SQL
	fields, values, placeholders := t.extractFieldsAndValues(model, false)

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		t.name, fields, placeholders)

	result, err := t.db.Exec(sql, values...)
	if err != nil {
		return fmt.Errorf("failed to create record: %w", err)
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

// Update updates an existing record in the table
func (t *gorTable) Update(model interface{}) error {
	// Validate model type
	if err := t.validateModel(model); err != nil {
		return err
	}

	// Call BeforeUpdate hook
	if hook, ok := model.(interface{ BeforeUpdate() error }); ok {
		if err := hook.BeforeUpdate(); err != nil {
			return err
		}
	}

	// Set updated timestamp
	setTimestamps(model, false)

	// Generate update SQL
	setParts, values := t.extractUpdateFields(model)
	id := getID(model)

	if id == nil {
		return fmt.Errorf("cannot update record without ID")
	}

	sql := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", t.name, setParts) // #nosec G201 - Table name is system-controlled
	values = append(values, id)

	result, err := t.db.Exec(sql, values...)
	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return fmt.Errorf("no record found with ID %v", id)
	}

	// Call AfterUpdate hook
	if hook, ok := model.(interface{ AfterUpdate() error }); ok {
		if err := hook.AfterUpdate(); err != nil {
			return err
		}
	}

	return nil
}

// Delete deletes a record by ID
func (t *gorTable) Delete(id interface{}) error {
	sql := fmt.Sprintf("DELETE FROM %s WHERE id = ?", t.name) // #nosec G201 - Table name is system-controlled
	result, err := t.db.Exec(sql, id)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return fmt.Errorf("no record found with ID %v", id)
	}

	return nil
}

// Find finds a record by ID
func (t *gorTable) Find(id interface{}, dest interface{}) error {
	sql := fmt.Sprintf("SELECT * FROM %s WHERE id = ?", t.name) // #nosec G201 - Table name is system-controlled
	row := t.db.QueryRow(sql, id)

	return t.scanRow(row, dest)
}

// BulkInsert inserts multiple records at once
func (t *gorTable) BulkInsert(models interface{}) error {
	v := reflect.ValueOf(models)
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("models must be a slice")
	}

	if v.Len() == 0 {
		return nil // No records to insert
	}

	// Get the first model to determine structure
	firstModel := v.Index(0).Interface()
	fields, _, placeholders := t.extractFieldsAndValues(firstModel, false)

	// Build bulk insert SQL
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES ", t.name, fields) // #nosec G201 - Table name is system-controlled

	var allValues []interface{}
	var valuePlaceholders []string

	for i := 0; i < v.Len(); i++ {
		model := v.Index(i).Interface()

		// Set timestamps for each model
		setTimestamps(model, true)

		_, values, _ := t.extractFieldsAndValues(model, false)
		allValues = append(allValues, values...)
		valuePlaceholders = append(valuePlaceholders, fmt.Sprintf("(%s)", placeholders))
	}

	sql += joinStrings(valuePlaceholders, ", ")

	_, err := t.db.Exec(sql, allValues...)
	if err != nil {
		return fmt.Errorf("failed to bulk insert: %w", err)
	}

	return nil
}

// BulkUpdate updates multiple records with the same values
func (t *gorTable) BulkUpdate(models interface{}) error {
	v := reflect.ValueOf(models)
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("models must be a slice")
	}

	if v.Len() == 0 {
		return nil
	}

	// For bulk update, we'll use a transaction to update each record
	tx, err := t.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback() // Ignore error, tx.Commit() will handle success case
	}()

	for i := 0; i < v.Len(); i++ {
		model := v.Index(i).Interface()

		// Set timestamps
		setTimestamps(model, false)

		// Generate update SQL
		setParts, values := t.extractUpdateFields(model)
		id := getID(model)

		if id == nil {
			return fmt.Errorf("cannot update record at index %d without ID", i)
		}

		sql := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", t.name, setParts) // #nosec G201 - Table name is system-controlled
		values = append(values, id)

		_, err := tx.Exec(sql, values...)
		if err != nil {
			return fmt.Errorf("failed to update record at index %d: %w", i, err)
		}
	}

	return tx.Commit()
}

// BulkDelete deletes multiple records by IDs
func (t *gorTable) BulkDelete(ids interface{}) error {
	v := reflect.ValueOf(ids)
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("ids must be a slice")
	}

	if v.Len() == 0 {
		return nil
	}

	// Build IN clause
	placeholders := make([]string, v.Len())
	values := make([]interface{}, v.Len())

	for i := 0; i < v.Len(); i++ {
		placeholders[i] = "?"
		values[i] = v.Index(i).Interface()
	}

	sql := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", // #nosec G201 - Table name is system-controlled
		t.name, joinStrings(placeholders, ", "))

	result, err := t.db.Exec(sql, values...)
	if err != nil {
		return fmt.Errorf("failed to bulk delete: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return fmt.Errorf("no records found with provided IDs")
	}

	return nil
}

// AddColumn adds a new column to the table
func (t *gorTable) AddColumn(column gor.Column) error {
	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
		t.name, column.Name, t.adapter.ColumnType(column))

	if !column.Nullable {
		sql += " NOT NULL"
	}

	if column.Default != nil {
		sql += fmt.Sprintf(" DEFAULT %v", column.Default)
	}

	_, err := t.db.Exec(sql)
	if err != nil {
		return fmt.Errorf("failed to add column: %w", err)
	}

	// Update local columns
	t.columns = append(t.columns, column)

	return nil
}

// DropColumn removes a column from the table
func (t *gorTable) DropColumn(name string) error {
	sql := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", t.name, name)

	_, err := t.db.Exec(sql)
	if err != nil {
		return fmt.Errorf("failed to drop column: %w", err)
	}

	// Update local columns
	for i, col := range t.columns {
		if col.Name == name {
			t.columns = append(t.columns[:i], t.columns[i+1:]...)
			break
		}
	}

	return nil
}

// AddIndex adds a new index to the table
func (t *gorTable) AddIndex(index gor.Index) error {
	sql := t.adapter.IndexSQL(index)

	_, err := t.db.Exec(sql)
	if err != nil {
		return fmt.Errorf("failed to add index: %w", err)
	}

	// Update local indexes
	t.indexes = append(t.indexes, index)

	return nil
}

// DropIndex removes an index from the table
func (t *gorTable) DropIndex(name string) error {
	sql := fmt.Sprintf("DROP INDEX %s", name)

	_, err := t.db.Exec(sql)
	if err != nil {
		return fmt.Errorf("failed to drop index: %w", err)
	}

	// Update local indexes
	for i, idx := range t.indexes {
		if idx.Name == name {
			t.indexes = append(t.indexes[:i], t.indexes[i+1:]...)
			break
		}
	}

	return nil
}

// Helper methods
func (t *gorTable) validateModel(model interface{}) error {
	if t.modelType == nil {
		return nil // Skip validation if no model type registered
	}

	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	if modelType != t.modelType {
		return fmt.Errorf("model type %s does not match table type %s",
			modelType.Name(), t.modelType.Name())
	}

	return nil
}

func (t *gorTable) extractFieldsAndValues(model interface{}, includeID bool) (string, []interface{}, string) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var fields []string
	var values []interface{}
	var placeholders []string

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		value := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip ID for inserts unless explicitly included
		if !includeID && field.Name == "ID" {
			continue
		}

		fieldName := toSnakeCase(field.Name)
		fields = append(fields, fieldName)
		values = append(values, value.Interface())
		placeholders = append(placeholders, "?")
	}

	return joinStrings(fields, ", "), values, joinStrings(placeholders, ", ")
}

func (t *gorTable) extractUpdateFields(model interface{}) (string, []interface{}) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var setParts []string
	var values []interface{}

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		value := v.Field(i)

		// Skip unexported fields and ID
		if !field.IsExported() || field.Name == "ID" {
			continue
		}

		fieldName := toSnakeCase(field.Name)
		setParts = append(setParts, fmt.Sprintf("%s = ?", fieldName))
		values = append(values, value.Interface())
	}

	return joinStrings(setParts, ", "), values
}

func (t *gorTable) scanRow(row *sql.Row, dest interface{}) error {
	// Get the fields of the destination struct
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer")
	}

	destValue = destValue.Elem()
	destType := destValue.Type()

	// Prepare scan destinations
	scanDests := make([]interface{}, destType.NumField())
	for i := 0; i < destType.NumField(); i++ {
		field := destValue.Field(i)
		if field.CanSet() {
			scanDests[i] = field.Addr().Interface()
		}
	}

	return row.Scan(scanDests...)
}

func extractIndexes(t reflect.Type, tableName string) []gor.Index {
	var indexes []gor.Index

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("gor")

		if contains(tag, "index") {
			index := gor.Index{
				Name:    fmt.Sprintf("idx_%s_%s", tableName, toSnakeCase(field.Name)),
				Table:   tableName,
				Columns: []string{toSnakeCase(field.Name)},
				Unique:  contains(tag, "unique"),
			}
			indexes = append(indexes, index)
		}
	}

	return indexes
}
