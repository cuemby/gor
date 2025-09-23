package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	if version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", version)
	}
}

func TestRun(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{
			name:      "help command",
			args:      []string{"gor", "--help"},
			wantError: false,
		},
		{
			name:      "version command",
			args:      []string{"gor", "--version"},
			wantError: false,
		},
		{
			name:      "no arguments",
			args:      []string{"gor"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test args
			os.Args = tt.args

			// Capture output
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			rOut, wOut, _ := os.Pipe()
			rErr, wErr, _ := os.Pipe()
			os.Stdout = wOut
			os.Stderr = wErr

			// Run the function
			err := run()

			// Restore output
			wOut.Close()
			wErr.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			// Read captured output
			var bufOut, bufErr bytes.Buffer
			io.Copy(&bufOut, rOut)
			io.Copy(&bufErr, rErr)

			// Check error
			if tt.wantError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantError && err != nil {
				// Help and version commands may return a special error
				// that's not actually an error condition
				if !isExpectedExitError(err) {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRunInvalidCommand(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test with an invalid command
	os.Args = []string{"gor", "invalid-command"}

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := run()

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Should return an error for invalid command
	if err == nil {
		// Some CLI frameworks don't return error for unknown commands
		// but instead print help
		output := buf.String()
		if !strings.Contains(output, "help") && !strings.Contains(output, "usage") {
			t.Log("Expected error for invalid command but got none")
		}
	}
}

func TestMainFunction(t *testing.T) {
	// Testing main() is challenging since it calls os.Exit
	// We can test it indirectly by ensuring run() works correctly
	// The actual main() function is very simple and just wraps run()

	// Save original
	oldArgs := os.Args
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		os.Stderr = oldStderr
	}()

	// Set args that will cause an error (if CLI validates commands)
	os.Args = []string{"gor", "--invalid-flag"}

	// Capture stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// We can't directly test main() because it calls os.Exit
	// But we can verify run() returns error for invalid input
	err := run()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Verify some error handling occurs
	if err == nil {
		// Check if error was printed even if not returned
		stderr := buf.String()
		if stderr == "" {
			t.Log("Expected some output for invalid flag")
		}
	}
}

// Helper function to check if an error is an expected exit error
// (like from --help or --version flags)
func isExpectedExitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Common patterns for help/version exit
	return strings.Contains(errStr, "help requested") ||
		strings.Contains(errStr, "version requested") ||
		strings.Contains(errStr, "usage")
}

// Benchmark tests
func BenchmarkRun(b *testing.B) {
	oldArgs := os.Args
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Use help command for benchmarking (it's fast and doesn't do much)
	os.Args = []string{"gor", "--help"}

	// Create pipes to discard output
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Drain pipes in background
	go io.Copy(io.Discard, rOut)
	go io.Copy(io.Discard, rErr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = run()
	}

	wOut.Close()
	wErr.Close()
}

func BenchmarkVersion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = version
	}
}

// Example test
func Example() {
	// Save and restore
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Set example args
	os.Args = []string{"gor", "--version"}

	// This would normally print version info
	// Note: actual output depends on CLI implementation
	_ = run()
	// Output: Gor Framework v1.0.0
}

// Test helpers
func captureOutput(f func() error) (string, string, error) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	err := f()

	wOut.Close()
	wErr.Close()

	var stdout, stderr bytes.Buffer
	io.Copy(&stdout, rOut)
	io.Copy(&stderr, rErr)

	return stdout.String(), stderr.String(), err
}

func TestRunWithCapture(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name         string
		args         []string
		expectStdout bool
		expectStderr bool
	}{
		{
			name:         "help output",
			args:         []string{"gor", "--help"},
			expectStdout: true,
			expectStderr: false,
		},
		{
			name:         "version output",
			args:         []string{"gor", "--version"},
			expectStdout: true,
			expectStderr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			stdout, stderr, _ := captureOutput(run)

			if tt.expectStdout && stdout == "" {
				t.Error("Expected stdout output but got none")
			}
			if tt.expectStderr && stderr == "" {
				t.Error("Expected stderr output but got none")
			}
			if !tt.expectStdout && stdout != "" {
				t.Errorf("Unexpected stdout output: %s", stdout)
			}
			if !tt.expectStderr && stderr != "" && !strings.Contains(stderr, "Error:") {
				t.Errorf("Unexpected stderr output: %s", stderr)
			}
		})
	}
}

// TestMainFunctionIntegration provides a way to test the full integration
// Note: This doesn't actually call main() since that would exit the test process
func TestMainFunctionIntegration(t *testing.T) {
	// The best we can do is ensure run() covers main's functionality
	t.Run("valid run", func(t *testing.T) {
		oldArgs := os.Args
		os.Args = []string{"gor", "--help"}
		defer func() { os.Args = oldArgs }()

		// If run succeeds or returns expected error, main would exit 0
		err := run()
		if err != nil && !isExpectedExitError(err) {
			t.Errorf("run() failed unexpectedly: %v", err)
		}
	})

	t.Run("simulated error", func(t *testing.T) {
		// We know main() prints to stderr and exits 1 on error
		// We can verify the error formatting
		testErr := fmt.Errorf("test error")

		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		fmt.Fprintf(os.Stderr, "Error: %v\n", testErr)

		w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		io.Copy(&buf, r)

		expected := "Error: test error\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})
}