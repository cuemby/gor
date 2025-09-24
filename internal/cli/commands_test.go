package cli

import (
	"io"
	"os"
	"strings"
	"testing"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	return string(out)
}

func TestServerCommand(t *testing.T) {
	cmd := NewServerCommand()

	t.Run("Properties", func(t *testing.T) {
		if cmd.Name() != "server" {
			t.Errorf("Expected name 'server', got %s", cmd.Name())
		}
		if !strings.Contains(cmd.Description(), "development server") {
			t.Error("Description should mention development server")
		}
		if !strings.Contains(cmd.Usage(), "gor server") {
			t.Error("Usage should contain 'gor server'")
		}
	})

	// Note: We can't fully test Run() as it executes external commands
	// but we can test argument parsing
	t.Run("PortArgument", func(t *testing.T) {
		// This would require mocking exec.Command which is complex
		// For now, we just verify the command structure
		args := []string{"-p", "8080"}
		// In a real test, we'd mock exec.Command and verify it receives PORT=8080
		_ = args
	})
}

func TestConsoleCommand(t *testing.T) {
	cmd := NewConsoleCommand()

	t.Run("Properties", func(t *testing.T) {
		if cmd.Name() != "console" {
			t.Errorf("Expected name 'console', got %s", cmd.Name())
		}
		if !strings.Contains(cmd.Description(), "interactive console") {
			t.Error("Description should mention interactive console")
		}
		if !strings.Contains(cmd.Usage(), "gor console") {
			t.Error("Usage should contain 'gor console'")
		}
	})

	t.Run("Run", func(t *testing.T) {
		output := captureOutput(func() {
			err := cmd.Run([]string{})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})

		if !strings.Contains(output, "Starting Gor console") {
			t.Error("Output should contain 'Starting Gor console'")
		}
		if !strings.Contains(output, "not yet implemented") {
			t.Error("Output should mention not yet implemented")
		}
	})
}

func TestMigrateCommand(t *testing.T) {
	cmd := NewMigrateCommand()

	t.Run("Properties", func(t *testing.T) {
		if cmd.Name() != "migrate" {
			t.Errorf("Expected name 'migrate', got %s", cmd.Name())
		}
		if !strings.Contains(cmd.Description(), "database migrations") {
			t.Error("Description should mention database migrations")
		}
		if !strings.Contains(cmd.Usage(), "gor db migrate") {
			t.Error("Usage should contain 'gor db migrate'")
		}
	})

	t.Run("RunUp", func(t *testing.T) {
		output := captureOutput(func() {
			err := cmd.Run([]string{})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})

		if !strings.Contains(output, "Running migrations") {
			t.Error("Output should contain 'Running migrations'")
		}
		if !strings.Contains(output, "completed") {
			t.Error("Output should mention completion")
		}
	})

	t.Run("RunDown", func(t *testing.T) {
		output := captureOutput(func() {
			err := cmd.Run([]string{"down"})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})

		if !strings.Contains(output, "Rolling back") {
			t.Error("Output should contain 'Rolling back'")
		}
		if !strings.Contains(output, "Rollback completed") {
			t.Error("Output should mention rollback completion")
		}
	})

	t.Run("RunStatus", func(t *testing.T) {
		output := captureOutput(func() {
			err := cmd.Run([]string{"status"})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})

		if !strings.Contains(output, "Migration Status") {
			t.Error("Output should contain 'Migration Status'")
		}
		if !strings.Contains(output, "applied") {
			t.Error("Output should show applied migrations")
		}
		if !strings.Contains(output, "pending") {
			t.Error("Output should show pending migrations")
		}
	})

	t.Run("RunUnknownAction", func(t *testing.T) {
		err := cmd.Run([]string{"invalid"})
		if err == nil {
			t.Error("Expected error for unknown action")
		}
		if !strings.Contains(err.Error(), "unknown action") {
			t.Errorf("Error should mention unknown action: %v", err)
		}
	})
}

func TestRoutesCommand(t *testing.T) {
	cmd := NewRoutesCommand()

	t.Run("Properties", func(t *testing.T) {
		if cmd.Name() != "routes" {
			t.Errorf("Expected name 'routes', got %s", cmd.Name())
		}
		if !strings.Contains(cmd.Description(), "Display all routes") {
			t.Error("Description should mention displaying routes")
		}
		if !strings.Contains(cmd.Usage(), "gor routes") {
			t.Error("Usage should contain 'gor routes'")
		}
	})

	t.Run("Run", func(t *testing.T) {
		output := captureOutput(func() {
			err := cmd.Run([]string{})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})

		// Check for route table headers
		if !strings.Contains(output, "Method") {
			t.Error("Output should contain 'Method' header")
		}
		if !strings.Contains(output, "Path") {
			t.Error("Output should contain 'Path' header")
		}
		if !strings.Contains(output, "Handler") {
			t.Error("Output should contain 'Handler' header")
		}

		// Check for sample routes
		if !strings.Contains(output, "GET") {
			t.Error("Output should contain GET routes")
		}
		if !strings.Contains(output, "POST") {
			t.Error("Output should contain POST routes")
		}
		if !strings.Contains(output, "UsersController") {
			t.Error("Output should contain controller names")
		}
	})
}

func TestTestCommand(t *testing.T) {
	cmd := NewTestCommand()

	t.Run("Properties", func(t *testing.T) {
		if cmd.Name() != "test" {
			t.Errorf("Expected name 'test', got %s", cmd.Name())
		}
		if !strings.Contains(cmd.Description(), "Run tests") {
			t.Error("Description should mention running tests")
		}
		if !strings.Contains(cmd.Usage(), "gor test") {
			t.Error("Usage should contain 'gor test'")
		}
	})

	// Note: We can't fully test Run() as it executes go test
	// but we can verify the basic flow
}

func TestBuildCommand(t *testing.T) {
	cmd := NewBuildCommand()

	t.Run("Properties", func(t *testing.T) {
		if cmd.Name() != "build" {
			t.Errorf("Expected name 'build', got %s", cmd.Name())
		}
		if !strings.Contains(cmd.Description(), "Build the application") {
			t.Error("Description should mention building application")
		}
		if !strings.Contains(cmd.Usage(), "gor build") {
			t.Error("Usage should contain 'gor build'")
		}
	})

	// Note: We can't fully test Run() as it executes go build
	// but we can test argument parsing
	t.Run("OutputArgument", func(t *testing.T) {
		// This would require mocking exec.Command
		args := []string{"-o", "custom/output"}
		// In a real test, we'd mock exec.Command and verify it receives the custom output path
		_ = args
	})
}

func TestDeployCommand(t *testing.T) {
	cmd := NewDeployCommand()

	t.Run("Properties", func(t *testing.T) {
		if cmd.Name() != "deploy" {
			t.Errorf("Expected name 'deploy', got %s", cmd.Name())
		}
		if !strings.Contains(cmd.Description(), "Deploy the application") {
			t.Error("Description should mention deploying application")
		}
		if !strings.Contains(cmd.Usage(), "gor deploy") {
			t.Error("Usage should contain 'gor deploy'")
		}
	})

	t.Run("RunProduction", func(t *testing.T) {
		output := captureOutput(func() {
			err := cmd.Run([]string{})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})

		if !strings.Contains(output, "Deploying to production") {
			t.Error("Output should mention deploying to production")
		}
		if !strings.Contains(output, "Successfully deployed") {
			t.Error("Output should mention successful deployment")
		}
		if !strings.Contains(output, "example.com") {
			t.Error("Output should show deployment URL")
		}
	})

	t.Run("RunStaging", func(t *testing.T) {
		output := captureOutput(func() {
			err := cmd.Run([]string{"staging"})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})

		if !strings.Contains(output, "Deploying to staging") {
			t.Error("Output should mention deploying to staging")
		}
		if !strings.Contains(output, "staging.example.com") {
			t.Error("Output should show staging URL")
		}
	})

	t.Run("DeploymentSteps", func(t *testing.T) {
		output := captureOutput(func() {
			err := cmd.Run([]string{})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})

		expectedSteps := []string{
			"Running tests",
			"Building application",
			"Compiling assets",
			"Uploading to server",
			"Running migrations",
			"Restarting services",
			"Warming cache",
		}

		for _, step := range expectedSteps {
			if !strings.Contains(output, step) {
				t.Errorf("Output should contain deployment step: %s", step)
			}
		}
	})
}

// Benchmark tests
func BenchmarkServerCommand_Run(b *testing.B) {
	cmd := NewServerCommand()

	// We can't actually run the server command as it starts a process
	// So we'll benchmark the command setup
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cmd.Name()
		_ = cmd.Description()
		_ = cmd.Usage()
	}
}

func BenchmarkMigrateCommand_Run(b *testing.B) {
	cmd := NewMigrateCommand()

	// Redirect output to avoid console spam
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		w.Close()
		os.Stdout = old
		io.ReadAll(r) // Drain the pipe
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd.Run([]string{"status"})
	}
}

func BenchmarkRoutesCommand_Run(b *testing.B) {
	cmd := NewRoutesCommand()

	// Redirect output to avoid console spam
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		w.Close()
		os.Stdout = old
		io.ReadAll(r) // Drain the pipe
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd.Run([]string{})
	}
}

func BenchmarkDeployCommand_Run(b *testing.B) {
	cmd := NewDeployCommand()

	// Redirect output to avoid console spam
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		w.Close()
		os.Stdout = old
		io.ReadAll(r) // Drain the pipe
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd.Run([]string{"staging"})
	}
}
