package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test NewCommand
func TestNewCommand(t *testing.T) {
	cmd := NewNewCommand()

	t.Run("Name", func(t *testing.T) {
		if cmd.Name() != "new" {
			t.Error("Expected name 'new'")
		}
	})

	t.Run("Description", func(t *testing.T) {
		if cmd.Description() == "" {
			t.Error("Description should not be empty")
		}
	})

	t.Run("Usage", func(t *testing.T) {
		if !strings.Contains(cmd.Usage(), "new") {
			t.Error("Usage should contain 'new'")
		}
	})
}

func TestNewCommand_Run(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	cmd := NewNewCommand()

	tests := []struct {
		name       string
		args       []string
		wantError  bool
		fileExists bool
	}{
		{
			name:      "no arguments",
			args:      []string{},
			wantError: true,
		},
		{
			name:       "directory exists",
			args:       []string{"existingapp"},
			wantError:  true,
			fileExists: true,
		},
		{
			name:      "create new app",
			args:      []string{"newapp"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For "directory exists" test, create the directory
			if tt.fileExists && len(tt.args) > 0 {
				os.Mkdir(tt.args[0], 0755)
			}

			err := cmd.Run(tt.args)
			if (err != nil) != tt.wantError {
				t.Errorf("Run() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError && len(tt.args) > 0 {
				// Check key files were created
				appName := tt.args[0]
				expectedFiles := []string{
					appName + "/go.mod",
					appName + "/main.go",
					appName + "/config/application.go",
					appName + "/config/routes.go",
					appName + "/app/controllers/application_controller.go",
					appName + "/app/models/application_record.go",
					appName + "/Gorfile",
					appName + "/README.md",
				}

				for _, file := range expectedFiles {
					if !FileExists(file) {
						t.Errorf("Expected file %s not created", file)
					}
				}
			}
		})
	}
}

func TestNewCommand_CreateDirectoryStructure(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	cmd := NewNewCommand()
	appPath := "testapp"

	err := cmd.createDirectoryStructure(appPath)
	if err != nil {
		t.Errorf("createDirectoryStructure() error = %v", err)
	}

	expectedDirs := []string{
		"app/controllers",
		"app/models",
		"app/views/layouts",
		"app/views/shared",
		"app/jobs",
		"app/mailers",
		"app/channels",
		"config/environments",
		"db/migrations",
		"db/seeds",
		"test/controllers",
		"test/models",
		"public",
		"log",
	}

	for _, dir := range expectedDirs {
		expectedPath := filepath.Join(appPath, dir)
		if !FileExists(expectedPath) {
			t.Errorf("Expected directory %s not created", dir)
		}
	}
}

func TestNewCommand_ContentGenerators(t *testing.T) {
	cmd := NewNewCommand()

	t.Run("goModContent", func(t *testing.T) {
		content := cmd.goModContent("testapp")
		if !strings.Contains(content, "module testapp") {
			t.Error("go.mod should contain module name")
		}
		if !strings.Contains(content, "github.com/cuemby/gor") {
			t.Error("go.mod should require gor framework")
		}
	})

	t.Run("mainGoContent", func(t *testing.T) {
		content := cmd.mainGoContent()
		if !strings.Contains(content, "func main()") {
			t.Error("main.go should contain main function")
		}
		if !strings.Contains(content, "app.Start") {
			t.Error("main.go should start the application")
		}
		if !strings.Contains(content, "config.NewApplication()") {
			t.Error("main.go should create new application")
		}
	})

	t.Run("applicationContent", func(t *testing.T) {
		content := cmd.applicationContent()
		if !strings.Contains(content, "type Application struct") {
			t.Error("application.go should contain Application struct")
		}
		if !strings.Contains(content, "func NewApplication()") {
			t.Error("application.go should have NewApplication function")
		}
		if !strings.Contains(content, "initializeDatabase") {
			t.Error("application.go should initialize database")
		}
		if !strings.Contains(content, "initializeRouter") {
			t.Error("application.go should initialize router")
		}
	})

	t.Run("routesContent", func(t *testing.T) {
		content := cmd.routesContent()
		if !strings.Contains(content, "LoadRoutes") {
			t.Error("routes.go should contain LoadRoutes function")
		}
		if !strings.Contains(content, "r.GET") {
			t.Error("routes.go should define GET routes")
		}
	})

	t.Run("databaseYmlContent", func(t *testing.T) {
		content := cmd.databaseYmlContent()
		if !strings.Contains(content, "development:") {
			t.Error("database.yml should contain development config")
		}
		if !strings.Contains(content, "test:") {
			t.Error("database.yml should contain test config")
		}
		if !strings.Contains(content, "production:") {
			t.Error("database.yml should contain production config")
		}
		if !strings.Contains(content, "adapter: sqlite3") {
			t.Error("database.yml should use sqlite3 adapter")
		}
	})

	t.Run("developmentEnvContent", func(t *testing.T) {
		content := cmd.developmentEnvContent()
		if !strings.Contains(content, "debug") {
			t.Error("development.go should contain debug setting")
		}
		if !strings.Contains(content, "hot_reload") {
			t.Error("development.go should contain hot_reload setting")
		}
	})

	t.Run("productionEnvContent", func(t *testing.T) {
		content := cmd.productionEnvContent()
		if !strings.Contains(content, "cache_enabled") {
			t.Error("production.go should contain cache_enabled setting")
		}
		if !strings.Contains(content, "\"debug\":           false") {
			t.Error("production.go should disable debug")
		}
	})

	t.Run("testEnvContent", func(t *testing.T) {
		content := cmd.testEnvContent()
		if !strings.Contains(content, "log_level") {
			t.Error("test.go should contain log_level setting")
		}
	})

	t.Run("applicationControllerContent", func(t *testing.T) {
		content := cmd.applicationControllerContent()
		if !strings.Contains(content, "type ApplicationController struct") {
			t.Error("application_controller.go should contain ApplicationController")
		}
		if !strings.Contains(content, "BeforeAction") {
			t.Error("application_controller.go should have BeforeAction method")
		}
		if !strings.Contains(content, "AfterAction") {
			t.Error("application_controller.go should have AfterAction method")
		}
	})

	t.Run("homeControllerContent", func(t *testing.T) {
		content := cmd.homeControllerContent()
		if !strings.Contains(content, "type HomeController struct") {
			t.Error("home_controller.go should contain HomeController")
		}
		if !strings.Contains(content, "HomeIndex") {
			t.Error("home_controller.go should have HomeIndex function")
		}
	})

	t.Run("applicationRecordContent", func(t *testing.T) {
		content := cmd.applicationRecordContent()
		if !strings.Contains(content, "type ApplicationRecord struct") {
			t.Error("application_record.go should contain ApplicationRecord")
		}
		if !strings.Contains(content, "ID") {
			t.Error("application_record.go should have ID field")
		}
		if !strings.Contains(content, "CreatedAt") {
			t.Error("application_record.go should have CreatedAt field")
		}
		if !strings.Contains(content, "UpdatedAt") {
			t.Error("application_record.go should have UpdatedAt field")
		}
	})

	t.Run("layoutContent", func(t *testing.T) {
		content := cmd.layoutContent()
		if !strings.Contains(content, "<!DOCTYPE html>") {
			t.Error("layout should contain DOCTYPE")
		}
		if !strings.Contains(content, "{{template \"content\" .}}") {
			t.Error("layout should render content template")
		}
		if !strings.Contains(content, "<title>") {
			t.Error("layout should have title tag")
		}
	})

	t.Run("homeViewContent", func(t *testing.T) {
		content := cmd.homeViewContent()
		if !strings.Contains(content, "{{define \"content\"}}") {
			t.Error("home view should define content template")
		}
		if !strings.Contains(content, "{{.title}}") {
			t.Error("home view should display title")
		}
		if !strings.Contains(content, "{{.message}}") {
			t.Error("home view should display message")
		}
	})

	t.Run("seedsContent", func(t *testing.T) {
		content := cmd.seedsContent()
		if !strings.Contains(content, "func Run()") {
			t.Error("seeds.go should have Run function")
		}
		if !strings.Contains(content, "Seeding database") {
			t.Error("seeds.go should log seeding")
		}
	})

	t.Run("gorfileContent", func(t *testing.T) {
		content := cmd.gorfileContent()
		if !strings.Contains(content, "app_name:") {
			t.Error("Gorfile should contain app_name")
		}
		if !strings.Contains(content, "server:") {
			t.Error("Gorfile should contain server settings")
		}
		if !strings.Contains(content, "database:") {
			t.Error("Gorfile should contain database settings")
		}
	})

	t.Run("gitignoreContent", func(t *testing.T) {
		content := cmd.gitignoreContent()
		if !strings.Contains(content, "*.db") {
			t.Error("gitignore should ignore database files")
		}
		if !strings.Contains(content, "/vendor/") {
			t.Error("gitignore should ignore vendor directory")
		}
		if !strings.Contains(content, "*.log") {
			t.Error("gitignore should ignore log files")
		}
		if !strings.Contains(content, ".env") {
			t.Error("gitignore should ignore .env files")
		}
	})

	t.Run("readmeContent", func(t *testing.T) {
		content := cmd.readmeContent("TestApp")
		if !strings.Contains(content, "TestApp") {
			t.Error("README should contain app name")
		}
		if !strings.Contains(content, "Getting Started") {
			t.Error("README should have getting started section")
		}
		if !strings.Contains(content, "gor server") {
			t.Error("README should explain how to start server")
		}
		if !strings.Contains(content, "gor generate") {
			t.Error("README should explain code generation")
		}
	})
}

func TestNewCommand_GenerateFiles(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create app directory structure first
	cmd := NewNewCommand()
	appPath := "testapp"
	appName := "testapp"

	// Manually create the missing directories
	os.MkdirAll(filepath.Join(appPath, "app/views/home"), 0755)
	cmd.createDirectoryStructure(appPath)

	err := cmd.generateFiles(appPath, appName)
	if err != nil {
		t.Errorf("generateFiles() error = %v", err)
	}

	// Check that all expected files were created
	expectedFiles := []string{
		"go.mod",
		"main.go",
		"config/application.go",
		"config/database.yml",
		"config/routes.go",
		"config/environments/development.go",
		"config/environments/production.go",
		"config/environments/test.go",
		"app/controllers/application_controller.go",
		"app/controllers/home_controller.go",
		"app/models/application_record.go",
		"app/views/layouts/application.html",
		"app/views/home/index.html",
		"db/seeds/seeds.go",
		"Gorfile",
		".gitignore",
		"README.md",
	}

	for _, file := range expectedFiles {
		fullPath := filepath.Join(appPath, file)
		if !FileExists(fullPath) {
			t.Errorf("Expected file %s not created", file)
		}
	}
}

// Benchmark tests
func BenchmarkNewCommand_Run(b *testing.B) {
	tempDir := b.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	cmd := NewNewCommand()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		appName := filepath.Join("benchapp", string(rune(i)))
		args := []string{appName}
		_ = cmd.Run(args)
	}
}

func BenchmarkNewCommand_CreateDirectoryStructure(b *testing.B) {
	tempDir := b.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	cmd := NewNewCommand()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		appPath := filepath.Join("benchapp", string(rune(i)))
		_ = cmd.createDirectoryStructure(appPath)
	}
}

func BenchmarkNewCommand_GenerateFiles(b *testing.B) {
	tempDir := b.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	cmd := NewNewCommand()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		appPath := filepath.Join("benchapp", string(rune(i)))
		appName := "benchapp"
		cmd.createDirectoryStructure(appPath)
		_ = cmd.generateFiles(appPath, appName)
	}
}
