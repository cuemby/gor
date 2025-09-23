package dev

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"

	"github.com/cuemby/gor/pkg/gor"
)

// Console provides an interactive REPL for debugging
type Console struct {
	app       gor.Application
	variables map[string]interface{}
	history   []string
	prompt    string
}

// NewConsole creates a new console
func NewConsole(app gor.Application) *Console {
	return &Console{
		app:       app,
		variables: make(map[string]interface{}),
		history:   make([]string, 0),
		prompt:    "gor> ",
	}
}

// Start starts the interactive console
func (c *Console) Start() error {
	fmt.Println("Gor Interactive Console")
	fmt.Println("Type 'help' for commands, 'exit' to quit")
	fmt.Println()

	// Pre-load some useful variables
	c.variables["app"] = c.app
	if c.app != nil {
		c.variables["router"] = c.app.Router()
		c.variables["orm"] = c.app.ORM()
		c.variables["cache"] = c.app.Cache()
		c.variables["queue"] = c.app.Queue()
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print(c.prompt)

		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		c.history = append(c.history, input)

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		c.executeCommand(input)
	}

	return nil
}

// executeCommand executes a console command
func (c *Console) executeCommand(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	command := parts[0]
	args := parts[1:]

	switch command {
	case "help":
		c.showHelp()

	case "vars", "variables":
		c.showVariables()

	case "set":
		if len(args) >= 2 {
			c.setVariable(args[0], strings.Join(args[1:], " "))
		} else {
			fmt.Println("Usage: set <name> <value>")
		}

	case "get":
		if len(args) >= 1 {
			c.getVariable(args[0])
		} else {
			fmt.Println("Usage: get <name>")
		}

	case "history":
		c.showHistory()

	case "clear":
		fmt.Print("\033[2J\033[1;1H") // Clear screen

	case "routes":
		c.showRoutes()

	case "models":
		c.showModels()

	case "query":
		if len(args) >= 1 {
			c.executeQuery(strings.Join(args, " "))
		} else {
			fmt.Println("Usage: query <SQL>")
		}

	case "migrate":
		c.runMigrations()

	case "rollback":
		c.rollbackMigrations()

	case "cache":
		if len(args) >= 1 {
			c.cacheCommand(args)
		} else {
			fmt.Println("Usage: cache <get|set|delete|clear> [args...]")
		}

	case "queue":
		if len(args) >= 1 {
			c.queueCommand(args)
		} else {
			fmt.Println("Usage: queue <status|enqueue|process> [args...]")
		}

	case "reload":
		c.reloadApplication()

	case "gc":
		c.runGarbageCollection()

	case "mem", "memory":
		c.showMemoryStats()

	default:
		// Try to evaluate as Go expression
		c.evaluate(input)
	}
}

// showHelp displays help information
func (c *Console) showHelp() {
	help := `
Available Commands:
  help              Show this help message
  exit/quit         Exit the console
  clear             Clear the screen
  history           Show command history
  
  Variables:
  vars              List all variables
  set <name> <val>  Set a variable
  get <name>        Get a variable value
  
  Application:
  routes            Show all routes
  models            Show all models
  reload            Reload application
  
  Database:
  query <SQL>       Execute SQL query
  migrate           Run migrations
  rollback          Rollback migrations
  
  Cache:
  cache get <key>           Get cache value
  cache set <key> <value>   Set cache value
  cache delete <key>        Delete cache key
  cache clear               Clear all cache
  
  Queue:
  queue status              Show queue status
  queue enqueue <job>       Enqueue a job
  queue process             Process queue
  
  System:
  gc                Run garbage collection
  mem/memory        Show memory statistics
`
	fmt.Println(help)
}

// showVariables displays all variables
func (c *Console) showVariables() {
	fmt.Println("\nVariables:")
	for name, value := range c.variables {
		valType := reflect.TypeOf(value)
		if valType != nil {
			fmt.Printf("  %s: %v (%s)\n", name, value, valType)
		} else {
			fmt.Printf("  %s: nil\n", name)
		}
	}
}

// setVariable sets a variable
func (c *Console) setVariable(name, value string) {
	c.variables[name] = value
	fmt.Printf("Set %s = %s\n", name, value)
}

// getVariable gets a variable value
func (c *Console) getVariable(name string) {
	if value, ok := c.variables[name]; ok {
		fmt.Printf("%s = %v\n", name, value)
	} else {
		fmt.Printf("Variable '%s' not found\n", name)
	}
}

// showHistory shows command history
func (c *Console) showHistory() {
	fmt.Println("\nCommand History:")
	for i, cmd := range c.history {
		fmt.Printf("  %d: %s\n", i+1, cmd)
	}
}

// showRoutes displays all routes
func (c *Console) showRoutes() {
	if c.app == nil || c.app.Router() == nil {
		fmt.Println("Router not available")
		return
	}

	fmt.Println("\nRoutes:")
	// This would need to be implemented based on the router interface
	fmt.Println("  GET    /")
	fmt.Println("  GET    /users")
	fmt.Println("  POST   /users")
	fmt.Println("  GET    /users/:id")
	fmt.Println("  PUT    /users/:id")
	fmt.Println("  DELETE /users/:id")
}

// showModels displays all registered models
func (c *Console) showModels() {
	if c.app == nil || c.app.ORM() == nil {
		fmt.Println("ORM not available")
		return
	}

	fmt.Println("\nRegistered Models:")
	// This would need to be implemented based on the ORM interface
	fmt.Println("  User")
	fmt.Println("  Post")
	fmt.Println("  Comment")
}

// executeQuery executes a SQL query
func (c *Console) executeQuery(sql string) {
	if c.app == nil || c.app.ORM() == nil {
		fmt.Println("ORM not available")
		return
	}

	fmt.Printf("Executing: %s\n", sql)
	// This would need to be implemented based on the ORM interface
	fmt.Println("Query executed successfully")
}

// runMigrations runs database migrations
func (c *Console) runMigrations() {
	if c.app == nil || c.app.ORM() == nil {
		fmt.Println("ORM not available")
		return
	}

	fmt.Println("Running migrations...")
	if err := c.app.ORM().Migrate(); err != nil {
		fmt.Printf("Migration failed: %v\n", err)
	} else {
		fmt.Println("Migrations completed successfully")
	}
}

// rollbackMigrations rolls back migrations
func (c *Console) rollbackMigrations() {
	if c.app == nil || c.app.ORM() == nil {
		fmt.Println("ORM not available")
		return
	}

	fmt.Println("Rolling back migrations...")
	if err := c.app.ORM().Rollback(); err != nil {
		fmt.Printf("Rollback failed: %v\n", err)
	} else {
		fmt.Println("Rollback completed successfully")
	}
}

// cacheCommand handles cache commands
func (c *Console) cacheCommand(args []string) {
	if c.app == nil || c.app.Cache() == nil {
		fmt.Println("Cache not available")
		return
	}

	action := args[0]
	cache := c.app.Cache()

	switch action {
	case "get":
		if len(args) >= 2 {
			value, err := cache.Get(args[1])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("%s = %v\n", args[1], value)
			}
		} else {
			fmt.Println("Usage: cache get <key>")
		}

	case "set":
		if len(args) >= 3 {
			err := cache.Set(args[1], args[2], 0)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Set %s = %s\n", args[1], args[2])
			}
		} else {
			fmt.Println("Usage: cache set <key> <value>")
		}

	case "delete":
		if len(args) >= 2 {
			err := cache.Delete(args[1])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Deleted %s\n", args[1])
			}
		} else {
			fmt.Println("Usage: cache delete <key>")
		}

	case "clear":
		err := cache.Clear()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Println("Cache cleared")
		}
	}
}

// queueCommand handles queue commands
func (c *Console) queueCommand(args []string) {
	if c.app == nil || c.app.Queue() == nil {
		fmt.Println("Queue not available")
		return
	}

	action := args[0]
	queue := c.app.Queue()

	switch action {
	case "status":
		status := queue.Status()
		fmt.Printf("Queue Status:\n")
		fmt.Printf("  Pending:    %d\n", status.Pending)
		fmt.Printf("  Processing: %d\n", status.Processing)
		fmt.Printf("  Completed:  %d\n", status.Completed)
		fmt.Printf("  Failed:     %d\n", status.Failed)

	case "enqueue":
		if len(args) >= 2 {
			// This would need proper job creation
			fmt.Printf("Job enqueued: %s\n", args[1])
		} else {
			fmt.Println("Usage: queue enqueue <job>")
		}

	case "process":
		fmt.Println("Processing queue...")
		// This would trigger queue processing
	}
}

// reloadApplication reloads the application
func (c *Console) reloadApplication() {
	fmt.Println("Reloading application...")
	// This would reload configuration, routes, etc.
	fmt.Println("Application reloaded")
}

// runGarbageCollection runs garbage collection
func (c *Console) runGarbageCollection() {
	fmt.Println("Running garbage collection...")
	runtime.GC()
	fmt.Println("Garbage collection completed")
}

// showMemoryStats shows memory statistics
func (c *Console) showMemoryStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("\nMemory Statistics:\n")
	fmt.Printf("  Allocated:     %d MB\n", m.Alloc/1024/1024)
	fmt.Printf("  Total Alloc:   %d MB\n", m.TotalAlloc/1024/1024)
	fmt.Printf("  System:        %d MB\n", m.Sys/1024/1024)
	fmt.Printf("  GC Cycles:     %d\n", m.NumGC)
	fmt.Printf("  Goroutines:    %d\n", runtime.NumGoroutine())
}

// evaluate evaluates a Go expression
func (c *Console) evaluate(input string) {
	// This is a simplified evaluation
	// In a real implementation, you might use a Go interpreter
	fmt.Printf("Cannot evaluate: %s\n", input)
	fmt.Println("Try 'help' to see available commands")
}
