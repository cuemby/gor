package orm

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/cuemby/gor/pkg/gor"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// SQLite adapter
type SQLiteAdapter struct{}

func NewSQLiteAdapter() gor.DatabaseAdapter {
	return &SQLiteAdapter{}
}

func (a *SQLiteAdapter) Connect(config gor.DatabaseConfig) (*sql.DB, error) {
	dsn := config.Database
	if dsn == "" {
		dsn = ":memory:" // Default to in-memory database
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// Enable foreign key support and WAL mode for better performance
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, err
	}

	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		// WAL mode might not be available in all SQLite builds, so we'll continue
		// without it rather than failing
	}

	return db, nil
}

func (a *SQLiteAdapter) Migrate(db *sql.DB, migrations []gor.Migration) error {
	// Create migrations table if it doesn't exist
	createMigrationsTable := `
		CREATE TABLE IF NOT EXISTS gor_migrations (
			version TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			sql TEXT NOT NULL
		)
	`
	if _, err := db.Exec(createMigrationsTable); err != nil {
		return err
	}

	// Apply migrations
	for _, migration := range migrations {
		// Check if migration already applied
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM gor_migrations WHERE version = ?", migration.Version).Scan(&count)
		if err != nil {
			return err
		}

		if count == 0 {
			// Apply migration
			if _, err := db.Exec(migration.SQL); err != nil {
				return fmt.Errorf("failed to apply migration %s: %w", migration.Version, err)
			}

			// Record migration
			_, err = db.Exec("INSERT INTO gor_migrations (version, name, sql) VALUES (?, ?, ?)",
				migration.Version, migration.Name, migration.SQL)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *SQLiteAdapter) GenerateSQL(qb gor.QueryBuilder) (string, []interface{}, error) {
	// This is a simplified implementation - would need to be expanded
	gorQB := qb.(*QueryBuilder)

	if gorQB.rawSQL != "" {
		return gorQB.rawSQL, gorQB.rawArgs, nil
	}

	var sql strings.Builder
	var args []interface{}

	if gorQB.isCount {
		sql.WriteString("SELECT COUNT(*) FROM ")
	} else {
		sql.WriteString("SELECT * FROM ")
	}

	sql.WriteString(gorQB.tableName)

	// Add JOINs
	for _, join := range gorQB.joins {
		sql.WriteString(" ")
		sql.WriteString(join)
	}

	// Add WHERE conditions
	if len(gorQB.whereConditions) > 0 {
		sql.WriteString(" WHERE ")
		sql.WriteString(strings.Join(gorQB.whereConditions, " AND "))
		args = append(args, gorQB.whereArgs...)
	}

	// Add ORDER BY
	if len(gorQB.orderBy) > 0 {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(strings.Join(gorQB.orderBy, ", "))
	}

	// Add LIMIT
	if gorQB.limitValue != nil {
		sql.WriteString(fmt.Sprintf(" LIMIT %d", *gorQB.limitValue))
	}

	// Add OFFSET
	if gorQB.offsetValue != nil {
		sql.WriteString(fmt.Sprintf(" OFFSET %d", *gorQB.offsetValue))
	}

	return sql.String(), args, nil
}

func (a *SQLiteAdapter) ColumnType(column gor.Column) string {
	switch column.Type {
	case "INTEGER":
		return "INTEGER"
	case "BIGINT":
		return "INTEGER"
	case "TEXT":
		if column.Size > 0 {
			return fmt.Sprintf("TEXT(%d)", column.Size)
		}
		return "TEXT"
	case "BOOLEAN":
		return "INTEGER" // SQLite uses INTEGER for boolean
	case "REAL":
		return "REAL"
	case "DOUBLE":
		return "REAL"
	case "BLOB":
		return "BLOB"
	case "TIMESTAMP":
		return "TIMESTAMP"
	default:
		return "TEXT"
	}
}

func (a *SQLiteAdapter) IndexSQL(index gor.Index) string {
	sql := "CREATE "
	if index.Unique {
		sql += "UNIQUE "
	}
	sql += fmt.Sprintf("INDEX %s ON %s (%s)",
		index.Name, index.Table, strings.Join(index.Columns, ", "))
	return sql
}

func (a *SQLiteAdapter) CreateTableSQL(tableName string, columns []gor.Column) string {
	var columnDefs []string

	for _, col := range columns {
		def := fmt.Sprintf("%s %s", col.Name, a.ColumnType(col))

		if col.PrimaryKey {
			def += " PRIMARY KEY AUTOINCREMENT"
		}

		if !col.Nullable && !col.PrimaryKey {
			def += " NOT NULL"
		}

		if col.Unique && !col.PrimaryKey {
			def += " UNIQUE"
		}

		if col.Default != nil {
			def += fmt.Sprintf(" DEFAULT %v", col.Default)
		}

		columnDefs = append(columnDefs, def)
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(columnDefs, ", "))
}

// PostgreSQL adapter
type PostgreSQLAdapter struct{}

func NewPostgreSQLAdapter() gor.DatabaseAdapter {
	return &PostgreSQLAdapter{}
}

func (a *PostgreSQLAdapter) Connect(config gor.DatabaseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.Username, config.Password, config.Database, config.SSLMode)

	return sql.Open("postgres", dsn)
}

func (a *PostgreSQLAdapter) Migrate(db *sql.DB, migrations []gor.Migration) error {
	// Similar to SQLite but with PostgreSQL-specific syntax
	createMigrationsTable := `
		CREATE TABLE IF NOT EXISTS gor_migrations (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			sql TEXT NOT NULL
		)
	`
	if _, err := db.Exec(createMigrationsTable); err != nil {
		return err
	}

	// Apply migrations (same logic as SQLite)
	for _, migration := range migrations {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM gor_migrations WHERE version = $1", migration.Version).Scan(&count)
		if err != nil {
			return err
		}

		if count == 0 {
			if _, err := db.Exec(migration.SQL); err != nil {
				return fmt.Errorf("failed to apply migration %s: %w", migration.Version, err)
			}

			_, err = db.Exec("INSERT INTO gor_migrations (version, name, sql) VALUES ($1, $2, $3)",
				migration.Version, migration.Name, migration.SQL)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *PostgreSQLAdapter) GenerateSQL(qb gor.QueryBuilder) (string, []interface{}, error) {
	// Similar to SQLite but with PostgreSQL-specific syntax (e.g., $1, $2 instead of ?)
	gorQB := qb.(*QueryBuilder)

	if gorQB.rawSQL != "" {
		return gorQB.rawSQL, gorQB.rawArgs, nil
	}

	var sql strings.Builder
	var args []interface{}

	if gorQB.isCount {
		sql.WriteString("SELECT COUNT(*) FROM ")
	} else {
		sql.WriteString("SELECT * FROM ")
	}

	sql.WriteString(gorQB.tableName)

	// Add JOINs
	for _, join := range gorQB.joins {
		sql.WriteString(" ")
		sql.WriteString(join)
	}

	// Add WHERE conditions with PostgreSQL parameter style
	if len(gorQB.whereConditions) > 0 {
		sql.WriteString(" WHERE ")
		// Convert ? placeholders to $1, $2, etc.
		conditions := make([]string, len(gorQB.whereConditions))
		paramIndex := 1
		for i, condition := range gorQB.whereConditions {
			// Simple replacement - would need more sophisticated parsing in production
			placeholderCount := strings.Count(condition, "?")
			for j := 0; j < placeholderCount; j++ {
				condition = strings.Replace(condition, "?", fmt.Sprintf("$%d", paramIndex), 1)
				paramIndex++
			}
			conditions[i] = condition
		}
		sql.WriteString(strings.Join(conditions, " AND "))
		args = append(args, gorQB.whereArgs...)
	}

	// Add ORDER BY
	if len(gorQB.orderBy) > 0 {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(strings.Join(gorQB.orderBy, ", "))
	}

	// Add LIMIT
	if gorQB.limitValue != nil {
		sql.WriteString(fmt.Sprintf(" LIMIT %d", *gorQB.limitValue))
	}

	// Add OFFSET
	if gorQB.offsetValue != nil {
		sql.WriteString(fmt.Sprintf(" OFFSET %d", *gorQB.offsetValue))
	}

	return sql.String(), args, nil
}

func (a *PostgreSQLAdapter) ColumnType(column gor.Column) string {
	switch column.Type {
	case "INTEGER":
		return "INTEGER"
	case "BIGINT":
		return "BIGINT"
	case "TEXT":
		if column.Size > 0 {
			return fmt.Sprintf("VARCHAR(%d)", column.Size)
		}
		return "TEXT"
	case "BOOLEAN":
		return "BOOLEAN"
	case "REAL":
		return "REAL"
	case "DOUBLE":
		return "DOUBLE PRECISION"
	case "BLOB":
		return "BYTEA"
	case "TIMESTAMP":
		return "TIMESTAMP"
	default:
		return "TEXT"
	}
}

func (a *PostgreSQLAdapter) IndexSQL(index gor.Index) string {
	sql := "CREATE "
	if index.Unique {
		sql += "UNIQUE "
	}
	sql += fmt.Sprintf("INDEX %s ON %s (%s)",
		index.Name, index.Table, strings.Join(index.Columns, ", "))
	return sql
}

func (a *PostgreSQLAdapter) CreateTableSQL(tableName string, columns []gor.Column) string {
	var columnDefs []string

	for _, col := range columns {
		def := fmt.Sprintf("%s %s", col.Name, a.ColumnType(col))

		if col.PrimaryKey {
			def += " PRIMARY KEY"
		}

		if !col.Nullable && !col.PrimaryKey {
			def += " NOT NULL"
		}

		if col.Unique && !col.PrimaryKey {
			def += " UNIQUE"
		}

		if col.Default != nil {
			def += fmt.Sprintf(" DEFAULT %v", col.Default)
		}

		columnDefs = append(columnDefs, def)
	}

	return fmt.Sprintf("CREATE TABLE %s (%s)", tableName, strings.Join(columnDefs, ", "))
}

// MySQL adapter
type MySQLAdapter struct{}

func NewMySQLAdapter() gor.DatabaseAdapter {
	return &MySQLAdapter{}
}

func (a *MySQLAdapter) Connect(config gor.DatabaseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		config.Username, config.Password, config.Host, config.Port, config.Database)

	return sql.Open("mysql", dsn)
}

func (a *MySQLAdapter) Migrate(db *sql.DB, migrations []gor.Migration) error {
	// Similar implementation for MySQL
	createMigrationsTable := `
		CREATE TABLE IF NOT EXISTS gor_migrations (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			sql TEXT NOT NULL
		)
	`
	if _, err := db.Exec(createMigrationsTable); err != nil {
		return err
	}

	for _, migration := range migrations {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM gor_migrations WHERE version = ?", migration.Version).Scan(&count)
		if err != nil {
			return err
		}

		if count == 0 {
			if _, err := db.Exec(migration.SQL); err != nil {
				return fmt.Errorf("failed to apply migration %s: %w", migration.Version, err)
			}

			_, err = db.Exec("INSERT INTO gor_migrations (version, name, sql) VALUES (?, ?, ?)",
				migration.Version, migration.Name, migration.SQL)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *MySQLAdapter) GenerateSQL(qb gor.QueryBuilder) (string, []interface{}, error) {
	// Similar to SQLite implementation
	gorQB := qb.(*QueryBuilder)

	if gorQB.rawSQL != "" {
		return gorQB.rawSQL, gorQB.rawArgs, nil
	}

	var sql strings.Builder
	var args []interface{}

	if gorQB.isCount {
		sql.WriteString("SELECT COUNT(*) FROM ")
	} else {
		sql.WriteString("SELECT * FROM ")
	}

	sql.WriteString(gorQB.tableName)

	// Add JOINs
	for _, join := range gorQB.joins {
		sql.WriteString(" ")
		sql.WriteString(join)
	}

	// Add WHERE conditions
	if len(gorQB.whereConditions) > 0 {
		sql.WriteString(" WHERE ")
		sql.WriteString(strings.Join(gorQB.whereConditions, " AND "))
		args = append(args, gorQB.whereArgs...)
	}

	// Add ORDER BY
	if len(gorQB.orderBy) > 0 {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(strings.Join(gorQB.orderBy, ", "))
	}

	// Add LIMIT
	if gorQB.limitValue != nil {
		sql.WriteString(fmt.Sprintf(" LIMIT %d", *gorQB.limitValue))
	}

	// Add OFFSET
	if gorQB.offsetValue != nil {
		sql.WriteString(fmt.Sprintf(" OFFSET %d", *gorQB.offsetValue))
	}

	return sql.String(), args, nil
}

func (a *MySQLAdapter) ColumnType(column gor.Column) string {
	switch column.Type {
	case "INTEGER":
		return "INT"
	case "BIGINT":
		return "BIGINT"
	case "TEXT":
		if column.Size > 0 {
			return fmt.Sprintf("VARCHAR(%d)", column.Size)
		}
		return "TEXT"
	case "BOOLEAN":
		return "BOOLEAN"
	case "REAL":
		return "FLOAT"
	case "DOUBLE":
		return "DOUBLE"
	case "BLOB":
		return "BLOB"
	case "TIMESTAMP":
		return "TIMESTAMP"
	default:
		return "TEXT"
	}
}

func (a *MySQLAdapter) IndexSQL(index gor.Index) string {
	sql := "CREATE "
	if index.Unique {
		sql += "UNIQUE "
	}
	sql += fmt.Sprintf("INDEX %s ON %s (%s)",
		index.Name, index.Table, strings.Join(index.Columns, ", "))
	return sql
}

func (a *MySQLAdapter) CreateTableSQL(tableName string, columns []gor.Column) string {
	var columnDefs []string

	for _, col := range columns {
		def := fmt.Sprintf("%s %s", col.Name, a.ColumnType(col))

		if col.PrimaryKey {
			def += " PRIMARY KEY"
		}

		if !col.Nullable && !col.PrimaryKey {
			def += " NOT NULL"
		}

		if col.Unique && !col.PrimaryKey {
			def += " UNIQUE"
		}

		if col.Default != nil {
			def += fmt.Sprintf(" DEFAULT %v", col.Default)
		}

		columnDefs = append(columnDefs, def)
	}

	return fmt.Sprintf("CREATE TABLE %s (%s)", tableName, strings.Join(columnDefs, ", "))
}