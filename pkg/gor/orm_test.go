package gor

import (
	"testing"
	"time"
)

// Test BaseModel implementation
type TestModel struct {
	BaseModel
	Name string
}

func TestBaseModel(t *testing.T) {
	model := &TestModel{}

	t.Run("ID", func(t *testing.T) {
		// Test SetID and GetID with uint type
		model.SetID(uint(123))
		if model.GetID() != uint(123) {
			t.Errorf("GetID() = %v, want 123", model.GetID())
		}
	})

	t.Run("Timestamps", func(t *testing.T) {
		now := time.Now()

		// Test CreatedAt
		model.SetCreatedAt(now)
		if model.GetCreatedAt() != now {
			t.Errorf("GetCreatedAt() = %v, want %v", model.GetCreatedAt(), now)
		}

		// Test UpdatedAt
		later := now.Add(time.Hour)
		model.SetUpdatedAt(later)
		if model.GetUpdatedAt() != later {
			t.Errorf("GetUpdatedAt() = %v, want %v", model.GetUpdatedAt(), later)
		}
	})

	// TableName test removed as TestModel doesn't have TableName method

	t.Run("Hooks", func(t *testing.T) {
		// All hooks should return nil by default
		if err := model.Validate(); err != nil {
			t.Errorf("Validate() error = %v", err)
		}
		if err := model.BeforeCreate(); err != nil {
			t.Errorf("BeforeCreate() error = %v", err)
		}
		if err := model.AfterCreate(); err != nil {
			t.Errorf("AfterCreate() error = %v", err)
		}
		if err := model.BeforeUpdate(); err != nil {
			t.Errorf("BeforeUpdate() error = %v", err)
		}
		if err := model.AfterUpdate(); err != nil {
			t.Errorf("AfterUpdate() error = %v", err)
		}
		if err := model.BeforeDelete(); err != nil {
			t.Errorf("BeforeDelete() error = %v", err)
		}
		if err := model.AfterDelete(); err != nil {
			t.Errorf("AfterDelete() error = %v", err)
		}
	})
}

func TestColumnStructure(t *testing.T) {
	column := Column{
		Name:       "test_column",
		Type:       "VARCHAR(255)",
		Default:    "default_value",
		PrimaryKey: true,
		Unique:     true,
		Index:      true,
	}

	if column.Name != "test_column" {
		t.Errorf("Expected column name 'test_column', got %s", column.Name)
	}
	if !column.PrimaryKey {
		t.Error("Column should be primary key")
	}
	// NotNull field doesn't exist in Column struct
	if !column.Unique {
		t.Error("Column should be unique")
	}
}

func TestIndexStructure(t *testing.T) {
	index := Index{
		Name:    "idx_users_email",
		Columns: []string{"email", "username"},
		Unique:  true,
	}

	if index.Name != "idx_users_email" {
		t.Errorf("Expected index name 'idx_users_email', got %s", index.Name)
	}
	if len(index.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(index.Columns))
	}
	if !index.Unique {
		t.Error("Index should be unique")
	}
}

func TestMigration(t *testing.T) {
	migration := Migration{
		Version: "20240101_create_users",
	}

	if migration.Version != "20240101_create_users" {
		t.Errorf("Expected version '20240101_create_users', got %s", migration.Version)
	}
	// MigratedAt field doesn't exist in Migration struct
}

func TestDatabaseConfig(t *testing.T) {
	config := DatabaseConfig{
		Driver:          "postgres",
		Host:            "localhost",
		Port:            5432,
		Username:        "testuser",
		Password:        "testpass",
		Database:        "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
	}

	if config.Driver != "postgres" {
		t.Errorf("Expected driver 'postgres', got %s", config.Driver)
	}
	if config.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", config.Port)
	}
	if config.MaxOpenConns != 25 {
		t.Errorf("Expected MaxOpenConns 25, got %d", config.MaxOpenConns)
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "email",
		Message: "invalid email format",
	}

	// ValidationError.Error() just returns the Message
	expected := "invalid email format"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}

// Benchmark tests
func BenchmarkBaseModel_SetGetID(b *testing.B) {
	model := &BaseModel{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.SetID(int64(i))
		_ = model.GetID()
	}
}

func BenchmarkBaseModel_Timestamps(b *testing.B) {
	model := &BaseModel{}
	now := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.SetCreatedAt(now)
		model.SetUpdatedAt(now)
		_ = model.GetCreatedAt()
		_ = model.GetUpdatedAt()
	}
}

func BenchmarkValidationError(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := &ValidationError{
			Field:   "email",
			Message: "invalid email format",
		}
		_ = err.Error()
	}
}