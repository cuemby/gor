package gor

import (
	"context"
	"database/sql"
	"reflect"
	"time"
)

// ORM defines the Object-Relational Mapping interface.
// Inspired by Rails ActiveRecord but designed for Go's type system.
type ORM interface {
	// Connection management
	Connect(ctx context.Context, config DatabaseConfig) error
	Close() error
	DB() *sql.DB

	// Migration management
	Migrate(ctx context.Context) error
	Rollback(ctx context.Context, steps int) error
	MigrationStatus(ctx context.Context) ([]Migration, error)

	// Model operations
	Register(models ...interface{}) error
	Table(name string) Table

	// Transaction support
	Transaction(ctx context.Context, fn func(tx Transaction) error) error

	// Query building
	Query(model interface{}) QueryBuilder
	Find(model interface{}, id interface{}) error
	FindAll(models interface{}) error
	Create(model interface{}) error
	Update(model interface{}) error
	Delete(model interface{}) error
}

// QueryBuilder provides a fluent interface for building database queries.
// Similar to Rails ActiveRecord query interface but type-safe.
type QueryBuilder interface {
	// Filtering
	Where(condition string, args ...interface{}) QueryBuilder
	WhereMap(conditions map[string]interface{}) QueryBuilder
	Not(condition string, args ...interface{}) QueryBuilder

	// Ordering and limiting
	Order(field string) QueryBuilder
	OrderDesc(field string) QueryBuilder
	Limit(limit int) QueryBuilder
	Offset(offset int) QueryBuilder

	// Pagination
	Page(page, size int) QueryBuilder

	// Joins and includes
	Joins(table string) QueryBuilder
	LeftJoin(table string) QueryBuilder
	Includes(associations ...string) QueryBuilder
	Preload(associations ...string) QueryBuilder

	// Aggregations
	Count() (int64, error)
	Sum(field string) (float64, error)
	Average(field string) (float64, error)
	Maximum(field string) (interface{}, error)
	Minimum(field string) (interface{}, error)

	// Execution
	First(dest interface{}) error
	Last(dest interface{}) error
	Find(dest interface{}) error
	FindAll(dest interface{}) error
	Exists() (bool, error)

	// Batch operations
	FindInBatches(dest interface{}, batchSize int, fn func(tx Transaction, batch interface{}) error) error
	UpdateAll(updates map[string]interface{}) (int64, error)
	DeleteAll() (int64, error)

	// Raw SQL
	Raw(sql string, args ...interface{}) QueryBuilder
}

// Table represents a database table with Active Record-style methods.
type Table interface {
	Name() string
	Columns() []Column
	Indexes() []Index

	// Table-level operations
	Create(model interface{}) error
	Update(model interface{}) error
	Delete(id interface{}) error
	Find(id interface{}, dest interface{}) error

	// Bulk operations
	BulkInsert(models interface{}) error
	BulkUpdate(models interface{}) error
	BulkDelete(ids interface{}) error

	// Schema operations
	AddColumn(column Column) error
	DropColumn(name string) error
	AddIndex(index Index) error
	DropIndex(name string) error
}

// Transaction provides transactional database operations.
type Transaction interface {
	// Transaction control
	Commit() error
	Rollback() error

	// Operations within transaction
	Create(model interface{}) error
	Update(model interface{}) error
	Delete(model interface{}) error
	Query(model interface{}) QueryBuilder

	// Raw SQL
	Exec(sql string, args ...interface{}) (sql.Result, error)
	QuerySQL(sql string, args ...interface{}) (*sql.Rows, error)
	QueryRow(sql string, args ...interface{}) *sql.Row
}

// Model defines the base interface for all ORM models.
type Model interface {
	// Primary key
	GetID() interface{}
	SetID(id interface{})

	// Timestamps
	GetCreatedAt() time.Time
	SetCreatedAt(t time.Time)
	GetUpdatedAt() time.Time
	SetUpdatedAt(t time.Time)

	// Validation
	Validate() error

	// Callbacks
	BeforeCreate() error
	AfterCreate() error
	BeforeUpdate() error
	AfterUpdate() error
	BeforeDelete() error
	AfterDelete() error

	// Table metadata
	TableName() string
}

// BaseModel provides a default implementation of common model functionality.
type BaseModel struct {
	ID        uint      `gor:"primary_key;auto_increment" json:"id"`
	CreatedAt time.Time `gor:"created_at" json:"created_at"`
	UpdatedAt time.Time `gor:"updated_at" json:"updated_at"`
}

func (m *BaseModel) GetID() interface{}       { return m.ID }
func (m *BaseModel) SetID(id interface{})     { m.ID = id.(uint) }
func (m *BaseModel) GetCreatedAt() time.Time  { return m.CreatedAt }
func (m *BaseModel) SetCreatedAt(t time.Time) { m.CreatedAt = t }
func (m *BaseModel) GetUpdatedAt() time.Time  { return m.UpdatedAt }
func (m *BaseModel) SetUpdatedAt(t time.Time) { m.UpdatedAt = t }
func (m *BaseModel) Validate() error          { return nil }
func (m *BaseModel) BeforeCreate() error      { return nil }
func (m *BaseModel) AfterCreate() error       { return nil }
func (m *BaseModel) BeforeUpdate() error      { return nil }
func (m *BaseModel) AfterUpdate() error       { return nil }
func (m *BaseModel) BeforeDelete() error      { return nil }
func (m *BaseModel) AfterDelete() error       { return nil }

// Migration represents a database migration.
type Migration struct {
	Version   string    `json:"version"`
	Name      string    `json:"name"`
	AppliedAt time.Time `json:"applied_at"`
	SQL       string    `json:"sql"`
}

// Column represents a database table column.
type Column struct {
	Name       string      `json:"name"`
	Type       string      `json:"type"`
	Size       int         `json:"size,omitempty"`
	Precision  int         `json:"precision,omitempty"`
	Scale      int         `json:"scale,omitempty"`
	Nullable   bool        `json:"nullable"`
	Default    interface{} `json:"default,omitempty"`
	PrimaryKey bool        `json:"primary_key"`
	Unique     bool        `json:"unique"`
	Index      bool        `json:"index"`
	Comment    string      `json:"comment,omitempty"`
}

// Index represents a database index.
type Index struct {
	Name    string   `json:"name"`
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Type    string   `json:"type,omitempty"` // btree, hash, gin, gist, etc.
}

// Association types for relationships between models
type Association interface {
	Type() AssociationType
	Model() reflect.Type
	ForeignKey() string
	References() string
}

type AssociationType int

const (
	BelongsTo AssociationType = iota
	HasOne
	HasMany
	ManyToMany
)

// Validation interface for model validation
type Validation interface {
	Validate(model interface{}) []ValidationError
}

type ValidationError struct {
	Field   string      `json:"field"`
	Tag     string      `json:"tag"`
	Message string      `json:"message"`
	Value   interface{} `json:"value"`
}

func (e ValidationError) Error() string {
	return e.Message
}

// Scope allows for reusable query logic
type Scope func(QueryBuilder) QueryBuilder

// Common scopes that can be used across models
var (
	// Recent returns records created in the last day
	Recent = func(qb QueryBuilder) QueryBuilder {
		return qb.Where("created_at > ?", time.Now().AddDate(0, 0, -1))
	}

	// Published returns records with published status
	Published = func(qb QueryBuilder) QueryBuilder {
		return qb.Where("published = ?", true)
	}
)

// DatabaseAdapter defines the interface for different database drivers
type DatabaseAdapter interface {
	Connect(config DatabaseConfig) (*sql.DB, error)
	Migrate(db *sql.DB, migrations []Migration) error
	GenerateSQL(query QueryBuilder) (string, []interface{}, error)
	ColumnType(column Column) string
	IndexSQL(index Index) string
	CreateTableSQL(tableName string, columns []Column) string
}
