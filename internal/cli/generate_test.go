package cli

import (
	"os"
	"strings"
	"testing"
)

// Mock tracking for testing - removed unused variables

// Test GenerateCommand
func TestGenerateCommand(t *testing.T) {
	cmd := NewGenerateCommand()

	t.Run("Name", func(t *testing.T) {
		if cmd.Name() != "generate" {
			t.Error("Expected name 'generate'")
		}
	})

	t.Run("Description", func(t *testing.T) {
		if cmd.Description() == "" {
			t.Error("Description should not be empty")
		}
	})

	t.Run("Usage", func(t *testing.T) {
		if !strings.Contains(cmd.Usage(), "generate") {
			t.Error("Usage should contain 'generate'")
		}
	})
}

func TestGenerateCommand_Run(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Logf("Failed to change back to old dir: %v", err)
		}
	}()

	cmd := NewGenerateCommand()

	tests := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{
			name:      "no arguments",
			args:      []string{},
			wantError: true,
		},
		{
			name:      "missing name",
			args:      []string{"model"},
			wantError: true,
		},
		{
			name:      "unknown generator",
			args:      []string{"unknown", "test"},
			wantError: true,
		},
		{
			name:      "scaffold generator",
			args:      []string{"scaffold", "Post", "title:string", "body:text"},
			wantError: false,
		},
		{
			name:      "model generator",
			args:      []string{"model", "User", "name:string", "email:string"},
			wantError: false,
		},
		{
			name:      "controller generator",
			args:      []string{"controller", "Posts", "index", "show"},
			wantError: false,
		},
		{
			name:      "migration generator",
			args:      []string{"migration", "create_users", "name:string"},
			wantError: false,
		},
		{
			name:      "job generator",
			args:      []string{"job", "EmailWorker"},
			wantError: false,
		},
		{
			name:      "mailer generator",
			args:      []string{"mailer", "UserMailer"},
			wantError: false,
		},
		{
			name:      "channel generator",
			args:      []string{"channel", "ChatChannel"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create necessary directories for file generation
			if !tt.wantError && len(tt.args) > 1 {
				if err := os.MkdirAll("app/models", 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll("app/controllers", 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll("app/jobs", 0755); err != nil {
					t.Fatal(err)
				}
				os.MkdirAll("app/mailers", 0755)
				os.MkdirAll("app/channels", 0755)
				os.MkdirAll("app/views", 0755)
				os.MkdirAll("db/migrations", 0755)

				// Create subdirectories for views in scaffold
				if tt.args[0] == "scaffold" {
					viewDir := "app/views/" + strings.ToLower(tt.args[1]) + "s"
					os.MkdirAll(viewDir, 0755)
				}
			}

			err := cmd.Run(tt.args)
			if (err != nil) != tt.wantError {
				t.Errorf("Run() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError && len(tt.args) > 1 {
				// Check files were created based on generator type
				switch tt.args[0] {
				case "model":
					if !FileExists("app/models/user.go") {
						t.Error("Model file not created")
					}
				case "controller":
					if !FileExists("app/controllers/posts_controller.go") {
						t.Error("Controller file not created")
					}
				case "job":
					if !FileExists("app/jobs/emailworker_job.go") {
						t.Error("Job file not created")
					}
				case "mailer":
					if !FileExists("app/mailers/usermailer_mailer.go") {
						t.Error("Mailer file not created")
					}
				case "channel":
					if !FileExists("app/channels/chatchannel_channel.go") {
						t.Error("Channel file not created")
					}
				}
			}
		})
	}
}

func TestGenerateCommand_ParseFields(t *testing.T) {
	cmd := &GenerateCommand{}

	tests := []struct {
		name   string
		fields []string
		want   int
	}{
		{
			name:   "simple fields",
			fields: []string{"name:string", "age:integer"},
			want:   2,
		},
		{
			name:   "fields with modifiers",
			fields: []string{"email:string:unique", "active:boolean:index"},
			want:   2,
		},
		{
			name:   "invalid field",
			fields: []string{"invalid"},
			want:   0,
		},
		{
			name:   "mixed valid and invalid",
			fields: []string{"name:string", "invalid", "age:integer"},
			want:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := cmd.parseFields(tt.fields)
			if len(parsed) != tt.want {
				t.Errorf("parseFields() returned %d fields, want %d", len(parsed), tt.want)
			}
		})
	}
}

func TestGenerateCommand_GoTypeToSQL(t *testing.T) {
	cmd := &GenerateCommand{}

	tests := []struct {
		goType string
		want   string
	}{
		{"string", "TEXT"},
		{"text", "TEXT"},
		{"integer", "INTEGER"},
		{"float", "REAL"},
		{"boolean", "BOOLEAN"},
		{"datetime", "TIMESTAMP"},
		{"date", "DATE"},
		{"time", "TIME"},
		{"unknown", "TEXT"},
	}

	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			if got := cmd.goTypeToSQL(tt.goType); got != tt.want {
				t.Errorf("goTypeToSQL(%s) = %s, want %s", tt.goType, got, tt.want)
			}
		})
	}
}

func TestGenerateCommand_ViewGeneration(t *testing.T) {
	cmd := &GenerateCommand{}

	tests := []struct {
		viewType string
		contains []string
	}{
		{
			"index",
			[]string{"<table", "Show", "Edit", "Delete"},
		},
		{
			"show",
			[]string{"Edit", "Back to List"},
		},
		{
			"new",
			[]string{"<form", "Create", "Cancel"},
		},
		{
			"edit",
			[]string{"<form", "Update", "Cancel"},
		},
		{
			"_form",
			[]string{"{{define \"form\"}}", "form-group"},
		},
	}

	fields := []string{"title:string", "body:text", "published:boolean"}

	for _, tt := range tests {
		t.Run(tt.viewType, func(t *testing.T) {
			content := cmd.generateViewContent("Post", tt.viewType, fields)
			for _, expected := range tt.contains {
				if !strings.Contains(content, expected) {
					t.Errorf("View %s should contain %s", tt.viewType, expected)
				}
			}
		})
	}
}

func TestGenerateCommand_ContentGenerators(t *testing.T) {
	cmd := &GenerateCommand{}

	t.Run("generateModelContent", func(t *testing.T) {
		fields := cmd.parseFields([]string{"name:string", "age:integer", "email:string:unique"})
		content := cmd.generateModelContent("User", fields)

		if !strings.Contains(content, "type User struct") {
			t.Error("Model content should contain type definition")
		}
		if !strings.Contains(content, "ApplicationRecord") {
			t.Error("Model should embed ApplicationRecord")
		}
		if !strings.Contains(content, "func (User) TableName()") {
			t.Error("Model should have TableName method")
		}
	})

	t.Run("generateFullControllerContent", func(t *testing.T) {
		content := cmd.generateFullControllerContent("PostController", "Post")

		if !strings.Contains(content, "type PostController struct") {
			t.Error("Controller content should contain type definition")
		}
		if !strings.Contains(content, "ApplicationController") {
			t.Error("Controller should embed ApplicationController")
		}
		if !strings.Contains(content, "func (c *PostController) Index") {
			t.Error("Controller should have Index method")
		}
		if !strings.Contains(content, "func (c *PostController) Show") {
			t.Error("Controller should have Show method")
		}
		if !strings.Contains(content, "func (c *PostController) Create") {
			t.Error("Controller should have Create method")
		}
	})

	t.Run("generateMigrationContent", func(t *testing.T) {
		// Test create table migration
		content := cmd.generateMigrationContent("create_users", []string{"name:string", "email:string"})
		if !strings.Contains(content, "CREATE TABLE") {
			t.Error("Create migration should contain CREATE TABLE")
		}

		// Test change migration
		content = cmd.generateMigrationContent("add_age_to_users", []string{})
		if !strings.Contains(content, "ALTER TABLE") {
			t.Error("Change migration should contain ALTER TABLE comment")
		}
	})

	t.Run("generateJobContent", func(t *testing.T) {
		content := cmd.generateJobContent("EmailWorker")
		if !strings.Contains(content, "type EmailWorker struct") {
			t.Error("Job content should contain type definition")
		}
		if !strings.Contains(content, "func (j *EmailWorker) Perform") {
			t.Error("Job should have Perform method")
		}
	})

	t.Run("generateMailerContent", func(t *testing.T) {
		content := cmd.generateMailerContent("UserMailer")
		if !strings.Contains(content, "type UserMailer struct") {
			t.Error("Mailer content should contain type definition")
		}
		if !strings.Contains(content, "BaseMailer") {
			t.Error("Mailer should embed BaseMailer")
		}
	})

	t.Run("generateChannelContent", func(t *testing.T) {
		content := cmd.generateChannelContent("ChatChannel")
		if !strings.Contains(content, "type ChatChannel struct") {
			t.Error("Channel content should contain type definition")
		}
		if !strings.Contains(content, "BaseChannel") {
			t.Error("Channel should embed BaseChannel")
		}
		if !strings.Contains(content, "Subscribe") {
			t.Error("Channel should have Subscribe method")
		}
	})
}

// Benchmark tests
func BenchmarkGenerateCommand_ParseFields(b *testing.B) {
	cmd := &GenerateCommand{}
	fields := []string{"name:string", "age:integer", "email:string:unique", "active:boolean:index"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cmd.parseFields(fields)
	}
}

func BenchmarkGenerateCommand_GenerateModel(b *testing.B) {
	// Create temp directory
	tempDir := b.TempDir()
	oldDir, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			b.Logf("Failed to change back to old dir: %v", err)
		}
	}()

	cmd := &GenerateCommand{}
	fields := []string{"name:string", "age:integer"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cmd.generateModel("User", fields)
	}
}
