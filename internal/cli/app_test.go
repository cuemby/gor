package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MockCommand for testing
type MockCommand struct {
	name        string
	description string
	usage       string
	runFunc     func(args []string) error
}

func (m *MockCommand) Name() string        { return m.name }
func (m *MockCommand) Description() string { return m.description }
func (m *MockCommand) Usage() string       { return m.usage }
func (m *MockCommand) Run(args []string) error {
	if m.runFunc != nil {
		return m.runFunc(args)
	}
	return nil
}

func TestNewApp(t *testing.T) {
	app := NewApp("1.0.0")

	if app == nil {
		t.Fatal("NewApp returned nil")
	}

	if app.version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", app.version)
	}

	// Check that default commands are registered
	expectedCommands := []string{
		"new", "generate", "server", "console",
		"migrate", "routes", "test", "build", "deploy",
	}

	for _, cmd := range expectedCommands {
		if _, exists := app.commands[cmd]; !exists {
			t.Errorf("Expected command %s to be registered", cmd)
		}
	}
}

func TestApp_RegisterCommand(t *testing.T) {
	app := &App{
		commands: make(map[string]Command),
	}

	mockCmd := &MockCommand{
		name:        "mock",
		description: "Mock command",
		usage:       "gor mock",
	}

	app.registerCommand(mockCmd)

	if cmd, exists := app.commands["mock"]; !exists {
		t.Error("Command was not registered")
	} else if cmd != mockCmd {
		t.Error("Registered command does not match")
	}
}

func TestApp_Run_Version(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"version command", []string{"gor", "version"}},
		{"short flag", []string{"gor", "-v"}},
		{"long flag", []string{"gor", "--version"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp("1.2.3")

			// Capture output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := app.Run(tt.args)

			w.Close()
			os.Stdout = old

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			output, _ := io.ReadAll(r)
			if !strings.Contains(string(output), "1.2.3") {
				t.Errorf("Version not in output: %s", output)
			}
		})
	}
}

func TestApp_Run_Help(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"help command", []string{"gor", "help"}},
		{"short flag", []string{"gor", "-h"}},
		{"long flag", []string{"gor", "--help"}},
		{"no args", []string{"gor"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp("1.0.0")

			// Capture output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := app.Run(tt.args)

			w.Close()
			os.Stdout = old

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			output, _ := io.ReadAll(r)
			outputStr := string(output)

			// Check for expected help content
			expectedStrings := []string{
				"Gor", "USAGE", "COMMANDS", "new", "generate",
			}

			for _, expected := range expectedStrings {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Help output missing expected string '%s'", expected)
				}
			}
		})
	}
}

func TestApp_Run_CommandHelp(t *testing.T) {
	app := NewApp("1.0.0")

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := app.Run([]string{"gor", "help", "generate"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	// Check that generate command help is shown
	if !strings.Contains(outputStr, "GENERATE") {
		t.Error("Command help should show command name in uppercase")
	}
	if !strings.Contains(outputStr, "USAGE:") {
		t.Error("Command help should show usage")
	}
}

func TestApp_Run_UnknownCommand(t *testing.T) {
	app := NewApp("1.0.0")

	err := app.Run([]string{"gor", "unknown"})

	if err == nil {
		t.Error("Expected error for unknown command")
	}

	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("Error message should mention unknown command: %v", err)
	}
}

func TestApp_Run_CommandShortcuts(t *testing.T) {
	shortcuts := map[string]string{
		"g":  "generate",
		"s":  "server",
		"c":  "console",
		"db": "migrate",
		"t":  "test",
	}

	for shortcut, fullCmd := range shortcuts {
		t.Run(shortcut, func(t *testing.T) {
			app := NewApp("1.0.0")

			// Create a mock command to verify it's called
			called := false
			mockCmd := &MockCommand{
				name: fullCmd,
				runFunc: func(args []string) error {
					called = true
					return nil
				},
			}
			app.commands[fullCmd] = mockCmd

			err := app.Run([]string{"gor", shortcut})

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !called {
				t.Errorf("Shortcut %s did not call %s command", shortcut, fullCmd)
			}
		})
	}
}

func TestApp_Run_CommandWithArgs(t *testing.T) {
	app := NewApp("1.0.0")

	var receivedArgs []string
	mockCmd := &MockCommand{
		name: "mock",
		runFunc: func(args []string) error {
			receivedArgs = args
			return nil
		},
	}
	app.commands["mock"] = mockCmd

	err := app.Run([]string{"gor", "mock", "arg1", "arg2", "arg3"})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(receivedArgs) != 3 {
		t.Errorf("Expected 3 args, got %d", len(receivedArgs))
	}

	expected := []string{"arg1", "arg2", "arg3"}
	for i, arg := range expected {
		if receivedArgs[i] != arg {
			t.Errorf("Expected arg[%d] = %s, got %s", i, arg, receivedArgs[i])
		}
	}
}

func TestApp_Run_CommandError(t *testing.T) {
	app := NewApp("1.0.0")

	mockCmd := &MockCommand{
		name: "error",
		runFunc: func(args []string) error {
			return fmt.Errorf("command failed")
		},
	}
	app.commands["error"] = mockCmd

	err := app.Run([]string{"gor", "error"})

	if err == nil {
		t.Error("Expected error from command")
	}

	if err.Error() != "command failed" {
		t.Errorf("Expected 'command failed', got %v", err)
	}
}

func TestApp_PrintCommandHelp_UnknownCommand(t *testing.T) {
	app := NewApp("1.0.0")

	err := app.printCommandHelp("nonexistent")

	if err == nil {
		t.Error("Expected error for unknown command")
	}

	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("Error should mention unknown command: %v", err)
	}
}

func TestFileExists(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test_file_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Test existing file
	if !FileExists(tmpFile.Name()) {
		t.Error("FileExists should return true for existing file")
	}

	// Test non-existing file
	if FileExists("/nonexistent/path/to/file") {
		t.Error("FileExists should return false for non-existing file")
	}

	// Test directory
	tmpDir, err := os.MkdirTemp("", "test_dir_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if !FileExists(tmpDir) {
		t.Error("FileExists should return true for existing directory")
	}
}

func TestCreateDirectory(t *testing.T) {
	tmpBase, err := os.MkdirTemp("", "test_base_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpBase)

	// Test creating new directory
	newDir := filepath.Join(tmpBase, "new", "nested", "dir")
	err = CreateDirectory(newDir)
	if err != nil {
		t.Errorf("Failed to create directory: %v", err)
	}

	if !FileExists(newDir) {
		t.Error("Directory was not created")
	}

	// Test creating existing directory (should not error)
	err = CreateDirectory(newDir)
	if err != nil {
		t.Errorf("CreateDirectory on existing directory should not error: %v", err)
	}
}

func TestWriteFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_write_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.txt")
	content := "Hello, Gor!"

	err = WriteFile(filePath, content)
	if err != nil {
		t.Errorf("Failed to write file: %v", err)
	}

	// Read and verify content
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}

	if string(data) != content {
		t.Errorf("Expected content '%s', got '%s'", content, string(data))
	}

	// Test overwriting file
	newContent := "Updated content"
	err = WriteFile(filePath, newContent)
	if err != nil {
		t.Errorf("Failed to overwrite file: %v", err)
	}

	data, err = os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Failed to read updated file: %v", err)
	}

	if string(data) != newContent {
		t.Errorf("Expected updated content '%s', got '%s'", newContent, string(data))
	}
}

// Benchmark tests
func BenchmarkApp_Run_Help(b *testing.B) {
	app := NewApp("1.0.0")

	// Redirect output to avoid console spam
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		w.Close()
		os.Stdout = old
		if _, err := io.ReadAll(r); err != nil {
			b.Logf("Failed to drain pipe: %v", err)
		} // Drain the pipe
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := app.Run([]string{"gor", "help"}); err != nil {
			b.Logf("Failed to run help: %v", err)
		}
	}
}

func BenchmarkApp_Run_Command(b *testing.B) {
	app := NewApp("1.0.0")

	mockCmd := &MockCommand{
		name: "bench",
		runFunc: func(args []string) error {
			// Simulate some work
			_ = len(args)
			return nil
		},
	}
	app.commands["bench"] = mockCmd

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := app.Run([]string{"gor", "bench", "arg1", "arg2"}); err != nil {
			b.Logf("Failed to run bench: %v", err)
		}
	}
}
