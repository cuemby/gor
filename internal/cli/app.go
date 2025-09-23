// Copyright (c) 2025 Cuemby
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package cli

import (
	"fmt"
	"os"
	"strings"
)

// App represents the CLI application
type App struct {
	version  string
	commands map[string]Command
}

// Command represents a CLI command
type Command interface {
	Name() string
	Description() string
	Usage() string
	Run(args []string) error
}

// NewApp creates a new CLI application
func NewApp(version string) *App {
	app := &App{
		version:  version,
		commands: make(map[string]Command),
	}

	// Register all commands
	app.registerCommand(NewNewCommand())
	app.registerCommand(NewGenerateCommand())
	app.registerCommand(NewServerCommand())
	app.registerCommand(NewConsoleCommand())
	app.registerCommand(NewMigrateCommand())
	app.registerCommand(NewRoutesCommand())
	app.registerCommand(NewTestCommand())
	app.registerCommand(NewBuildCommand())
	app.registerCommand(NewDeployCommand())

	return app
}

// registerCommand registers a command with the app
func (a *App) registerCommand(cmd Command) {
	a.commands[cmd.Name()] = cmd
}

// Run executes the CLI application
func (a *App) Run(args []string) error {
	if len(args) < 2 {
		a.printHelp()
		return nil
	}

	command := args[1]

	// Handle special cases
	switch command {
	case "version", "-v", "--version":
		fmt.Printf("Gor Framework v%s\n", a.version)
		return nil
	case "help", "-h", "--help":
		if len(args) > 2 {
			return a.printCommandHelp(args[2])
		}
		a.printHelp()
		return nil
	}

	// Find and execute command
	cmd, exists := a.commands[command]
	if !exists {
		// Check for shortcuts
		shortcuts := map[string]string{
			"g":  "generate",
			"s":  "server",
			"c":  "console",
			"db": "migrate",
			"t":  "test",
		}

		if fullCommand, ok := shortcuts[command]; ok {
			cmd = a.commands[fullCommand]
		} else {
			return fmt.Errorf("unknown command: %s\nRun 'gor help' for usage", command)
		}
	}

	return cmd.Run(args[2:])
}

// printHelp prints the general help message
func (a *App) printHelp() {
	fmt.Print(`
🚀 Gor - The Rails-Inspired Go Web Framework

USAGE:
  gor <command> [arguments]

COMMON COMMANDS:
  new <app>        Create a new Gor application
  generate <type>  Generate code (controller, model, scaffold, etc.)
  server           Start the development server
  console          Start an interactive console
  migrate          Run database migrations
  routes           Display all routes
  test             Run tests
  build            Build the application
  deploy           Deploy the application

SHORTCUTS:
  g  = generate
  s  = server
  c  = console
  db = migrate
  t  = test

EXAMPLES:
  gor new myapp                    Create a new application
  gor g scaffold Post title:string Generate a complete resource
  gor s                            Start the development server
  gor db migrate                   Run pending migrations
  gor g controller Users index show Create a controller

For more help on a command, run:
  gor help <command>
`)
}

// printCommandHelp prints help for a specific command
func (a *App) printCommandHelp(cmdName string) error {
	cmd, exists := a.commands[cmdName]
	if !exists {
		return fmt.Errorf("unknown command: %s", cmdName)
	}

	fmt.Printf("\n%s - %s\n\n", strings.ToUpper(cmd.Name()), cmd.Description())
	fmt.Println("USAGE:")
	fmt.Printf("  %s\n\n", cmd.Usage())

	return nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// CreateDirectory creates a directory if it doesn't exist
func CreateDirectory(path string) error {
	if !FileExists(path) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// WriteFile writes content to a file
func WriteFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
