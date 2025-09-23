package orm

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/cuemby/gor/pkg/gor"
)

// Migrator handles database migrations
type Migrator struct {
	db      *sql.DB
	adapter gor.DatabaseAdapter
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *sql.DB, adapter gor.DatabaseAdapter) *Migrator {
	return &Migrator{
		db:      db,
		adapter: adapter,
	}
}

// Migrate runs all pending migrations
func (m *Migrator) Migrate(ctx context.Context) error {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get all migrations
	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	// Apply each migration
	for _, migration := range migrations {
		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.Version, err)
		}
	}

	return nil
}

// Rollback rolls back the specified number of migrations
func (m *Migrator) Rollback(ctx context.Context, steps int) error {
	// Get applied migrations in reverse order
	appliedMigrations, err := m.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	if len(appliedMigrations) == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	// Sort by version descending
	sort.Slice(appliedMigrations, func(i, j int) bool {
		return appliedMigrations[i].Version > appliedMigrations[j].Version
	})

	// Limit to requested steps
	if steps > len(appliedMigrations) {
		steps = len(appliedMigrations)
	}

	toRollback := appliedMigrations[:steps]

	// Rollback each migration
	for _, migration := range toRollback {
		if err := m.rollbackMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to rollback migration %s: %w", migration.Version, err)
		}
	}

	return nil
}

// Status returns the current migration status
func (m *Migrator) Status(ctx context.Context) ([]gor.Migration, error) {
	appliedMigrations, err := m.getAppliedMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	return appliedMigrations, nil
}

// createMigrationsTable creates the migrations tracking table
func (m *Migrator) createMigrationsTable() error {
	sql := `
		CREATE TABLE IF NOT EXISTS gor_migrations (
			version TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			sql TEXT NOT NULL
		)
	`

	_, err := m.db.Exec(sql)
	return err
}

// loadMigrations loads all migration files from the migrations directory
func (m *Migrator) loadMigrations() ([]gor.Migration, error) {
	// In a real implementation, this would read migration files from disk
	// For now, we'll return an empty slice as we haven't implemented
	// the file system migration loading yet
	return []gor.Migration{}, nil
}

// applyMigration applies a single migration
func (m *Migrator) applyMigration(ctx context.Context, migration gor.Migration) error {
	// Check if migration is already applied
	var count int
	err := m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM gor_migrations WHERE version = ?", migration.Version).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		// Migration already applied
		return nil
	}

	// Start transaction
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Apply migration SQL
	if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration
	_, err = tx.ExecContext(ctx, "INSERT INTO gor_migrations (version, name, sql) VALUES (?, ?, ?)",
		migration.Version, migration.Name, migration.SQL)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// rollbackMigration rolls back a single migration
func (m *Migrator) rollbackMigration(ctx context.Context, migration gor.Migration) error {
	// Start transaction
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// For a real implementation, we would need to store rollback SQL
	// or generate it based on the forward migration
	// For now, we'll just remove the record from migrations table
	_, err = tx.ExecContext(ctx, "DELETE FROM gor_migrations WHERE version = ?", migration.Version)
	if err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	return tx.Commit()
}

// getAppliedMigrations returns all applied migrations
func (m *Migrator) getAppliedMigrations() ([]gor.Migration, error) {
	rows, err := m.db.Query("SELECT version, name, applied_at, sql FROM gor_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []gor.Migration
	for rows.Next() {
		var migration gor.Migration
		err := rows.Scan(&migration.Version, &migration.Name, &migration.AppliedAt, &migration.SQL)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, migration)
	}

	return migrations, rows.Err()
}

// MigrationGenerator helps generate new migration files
type MigrationGenerator struct {
	migrationsDir string
}

// NewMigrationGenerator creates a new migration generator
func NewMigrationGenerator(migrationsDir string) *MigrationGenerator {
	return &MigrationGenerator{
		migrationsDir: migrationsDir,
	}
}

// GenerateMigration creates a new migration file
func (mg *MigrationGenerator) GenerateMigration(name string) error {
	// Generate version based on timestamp
	version := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.sql", version, name)
	filepath := filepath.Join(mg.migrationsDir, filename)

	// Migration template
	template := fmt.Sprintf(`-- Migration: %s
-- Created: %s

-- +migrate Up

-- +migrate Down
`, name, time.Now().Format("2006-01-02 15:04:05"))

	// In a real implementation, we would write this to a file
	_ = filepath
	_ = template

	return nil
}

// CreateTableMigration generates a CREATE TABLE migration
func (mg *MigrationGenerator) CreateTableMigration(tableName string, columns []gor.Column) gor.Migration {
	version := time.Now().Format("20060102150405")
	name := fmt.Sprintf("create_%s_table", tableName)

	// Generate CREATE TABLE SQL
	var columnDefs []string
	for _, col := range columns {
		def := fmt.Sprintf("%s %s", col.Name, col.Type)

		if col.PrimaryKey {
			def += " PRIMARY KEY"
		}

		if !col.Nullable {
			def += " NOT NULL"
		}

		if col.Unique {
			def += " UNIQUE"
		}

		if col.Default != nil {
			def += fmt.Sprintf(" DEFAULT %v", col.Default)
		}

		columnDefs = append(columnDefs, def)
	}

	sql := fmt.Sprintf("CREATE TABLE %s (\n  %s\n);", tableName, joinStrings(columnDefs, ",\n  "))

	return gor.Migration{
		Version: version,
		Name:    name,
		SQL:     sql,
	}
}

// AddColumnMigration generates an ADD COLUMN migration
func (mg *MigrationGenerator) AddColumnMigration(tableName string, column gor.Column) gor.Migration {
	version := time.Now().Format("20060102150405")
	name := fmt.Sprintf("add_%s_to_%s", column.Name, tableName)

	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, column.Name, column.Type)

	if !column.Nullable {
		sql += " NOT NULL"
	}

	if column.Default != nil {
		sql += fmt.Sprintf(" DEFAULT %v", column.Default)
	}

	sql += ";"

	return gor.Migration{
		Version: version,
		Name:    name,
		SQL:     sql,
	}
}

// DropColumnMigration generates a DROP COLUMN migration
func (mg *MigrationGenerator) DropColumnMigration(tableName, columnName string) gor.Migration {
	version := time.Now().Format("20060102150405")
	name := fmt.Sprintf("drop_%s_from_%s", columnName, tableName)

	sql := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tableName, columnName)

	return gor.Migration{
		Version: version,
		Name:    name,
		SQL:     sql,
	}
}

// AddIndexMigration generates a CREATE INDEX migration
func (mg *MigrationGenerator) AddIndexMigration(index gor.Index) gor.Migration {
	version := time.Now().Format("20060102150405")
	name := fmt.Sprintf("add_index_%s", index.Name)

	sql := "CREATE "
	if index.Unique {
		sql += "UNIQUE "
	}
	sql += fmt.Sprintf("INDEX %s ON %s (%s);", index.Name, index.Table, joinStrings(index.Columns, ", "))

	return gor.Migration{
		Version: version,
		Name:    name,
		SQL:     sql,
	}
}

// DropIndexMigration generates a DROP INDEX migration
func (mg *MigrationGenerator) DropIndexMigration(indexName string) gor.Migration {
	version := time.Now().Format("20060102150405")
	name := fmt.Sprintf("drop_index_%s", indexName)

	sql := fmt.Sprintf("DROP INDEX %s;", indexName)

	return gor.Migration{
		Version: version,
		Name:    name,
		SQL:     sql,
	}
}
