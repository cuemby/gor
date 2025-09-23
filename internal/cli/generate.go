package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// GenerateCommand handles code generation
type GenerateCommand struct{}

// NewGenerateCommand creates a new generate command
func NewGenerateCommand() *GenerateCommand {
	return &GenerateCommand{}
}

// Name returns the command name
func (c *GenerateCommand) Name() string {
	return "generate"
}

// Description returns the command description
func (c *GenerateCommand) Description() string {
	return "Generate code (scaffold, model, controller, etc.)"
}

// Usage returns the command usage
func (c *GenerateCommand) Usage() string {
	return `gor generate <generator> <name> [fields...]

Generators:
  scaffold    Generate model, controller, views, and migration
  model       Generate a model and migration
  controller  Generate a controller and views
  migration   Generate a migration file
  job         Generate a background job
  mailer      Generate a mailer
  channel     Generate a WebSocket channel

Field format: name:type[:modifier]
  Types: string, text, integer, float, boolean, datetime, date, time, references
  Modifiers: index, unique, required`
}

// Run executes the command
func (c *GenerateCommand) Run(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("generator type and name are required")
	}

	generatorType := args[0]
	name := args[1]
	fields := args[2:]

	switch generatorType {
	case "scaffold":
		return c.generateScaffold(name, fields)
	case "model":
		return c.generateModel(name, fields)
	case "controller":
		return c.generateController(name, fields)
	case "migration":
		return c.generateMigration(name, fields)
	case "job":
		return c.generateJob(name)
	case "mailer":
		return c.generateMailer(name)
	case "channel":
		return c.generateChannel(name)
	default:
		return fmt.Errorf("unknown generator: %s", generatorType)
	}
}

func (c *GenerateCommand) generateScaffold(name string, fields []string) error {
	fmt.Printf("🏭 Generating scaffold for %s...\n\n", name)

	// Generate model
	if err := c.generateModel(name, fields); err != nil {
		return err
	}

	// Generate controller with all actions
	if err := c.generateFullController(name); err != nil {
		return err
	}

	// Generate views
	if err := c.generateViews(name, fields); err != nil {
		return err
	}

	// Update routes
	if err := c.addResourceRoute(name); err != nil {
		return err
	}

	fmt.Printf(`
✅ Scaffold generated successfully!

Don't forget to:
  1. Run migrations: gor db migrate
  2. Restart the server
  3. Visit http://localhost:3000/%s
`, strings.ToLower(name)+"s")

	return nil
}

func (c *GenerateCommand) generateModel(name string, fields []string) error {
	modelName := strings.Title(name)
	tableName := strings.ToLower(name) + "s"

	// Parse fields
	parsedFields := c.parseFields(fields)

	// Generate model file
	modelPath := filepath.Join("app/models", strings.ToLower(name)+".go")
	modelContent := c.generateModelContent(modelName, parsedFields)

	if err := WriteFile(modelPath, modelContent); err != nil {
		return err
	}
	fmt.Printf("  ✓ Created model: %s\n", modelPath)

	// Generate migration
	migrationName := fmt.Sprintf("create_%s", tableName)
	if err := c.generateMigration(migrationName, fields); err != nil {
		return err
	}

	return nil
}

func (c *GenerateCommand) generateController(name string, actions []string) error {
	controllerName := strings.Title(name) + "Controller"
	controllerPath := filepath.Join("app/controllers", strings.ToLower(name)+"_controller.go")

	if len(actions) == 0 {
		actions = []string{"index", "show"}
	}

	controllerContent := c.generateControllerContent(controllerName, name, actions)

	if err := WriteFile(controllerPath, controllerContent); err != nil {
		return err
	}
	fmt.Printf("  ✓ Created controller: %s\n", controllerPath)

	return nil
}

func (c *GenerateCommand) generateFullController(name string) error {
	controllerName := strings.Title(name) + "Controller"
	controllerPath := filepath.Join("app/controllers", strings.ToLower(name)+"_controller.go")

	controllerContent := c.generateFullControllerContent(controllerName, name)

	if err := WriteFile(controllerPath, controllerContent); err != nil {
		return err
	}
	fmt.Printf("  ✓ Created controller: %s\n", controllerPath)

	return nil
}

func (c *GenerateCommand) generateMigration(name string, fields []string) error {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	migrationPath := filepath.Join("db/migrations", timestamp+"_"+name+".go")

	migrationContent := c.generateMigrationContent(name, fields)

	if err := WriteFile(migrationPath, migrationContent); err != nil {
		return err
	}
	fmt.Printf("  ✓ Created migration: %s\n", migrationPath)

	return nil
}

func (c *GenerateCommand) generateViews(name string, fields []string) error {
	viewsDir := filepath.Join("app/views", strings.ToLower(name)+"s")
	if err := CreateDirectory(viewsDir); err != nil {
		return err
	}

	views := []string{"index", "show", "new", "edit", "_form"}
	for _, view := range views {
		viewPath := filepath.Join(viewsDir, view+".html")
		viewContent := c.generateViewContent(name, view, fields)

		if err := WriteFile(viewPath, viewContent); err != nil {
			return err
		}
		fmt.Printf("  ✓ Created view: %s\n", viewPath)
	}

	return nil
}

func (c *GenerateCommand) generateJob(name string) error {
	jobName := strings.Title(name) + "Job"
	jobPath := filepath.Join("app/jobs", strings.ToLower(name)+"_job.go")

	jobContent := c.generateJobContent(jobName)

	if err := WriteFile(jobPath, jobContent); err != nil {
		return err
	}
	fmt.Printf("  ✓ Created job: %s\n", jobPath)

	return nil
}

func (c *GenerateCommand) generateMailer(name string) error {
	mailerName := strings.Title(name) + "Mailer"
	mailerPath := filepath.Join("app/mailers", strings.ToLower(name)+"_mailer.go")

	mailerContent := c.generateMailerContent(mailerName)

	if err := WriteFile(mailerPath, mailerContent); err != nil {
		return err
	}
	fmt.Printf("  ✓ Created mailer: %s\n", mailerPath)

	return nil
}

func (c *GenerateCommand) generateChannel(name string) error {
	channelName := strings.Title(name) + "Channel"
	channelPath := filepath.Join("app/channels", strings.ToLower(name)+"_channel.go")

	channelContent := c.generateChannelContent(channelName)

	if err := WriteFile(channelPath, channelContent); err != nil {
		return err
	}
	fmt.Printf("  ✓ Created channel: %s\n", channelPath)

	return nil
}

func (c *GenerateCommand) addResourceRoute(name string) error {
	// This would update the routes file
	fmt.Printf("  ℹ️ Add this to config/routes.go:\n")
	fmt.Printf("     r.Resources(\"%ss\", &controllers.%sController{})\n",
		strings.ToLower(name), strings.Title(name))
	return nil
}

// Field parsing
type Field struct {
	Name      string
	Type      string
	GoType    string
	Modifiers []string
}

func (c *GenerateCommand) parseFields(fields []string) []Field {
	var parsed []Field

	for _, field := range fields {
		parts := strings.Split(field, ":")
		if len(parts) < 2 {
			continue
		}

		f := Field{
			Name: parts[0],
			Type: parts[1],
		}

		if len(parts) > 2 {
			f.Modifiers = parts[2:]
		}

		// Map to Go type
		switch f.Type {
		case "string", "text":
			f.GoType = "string"
		case "integer":
			f.GoType = "int"
		case "float":
			f.GoType = "float64"
		case "boolean":
			f.GoType = "bool"
		case "datetime", "date", "time":
			f.GoType = "time.Time"
		case "references":
			f.GoType = "uint"
		default:
			f.GoType = "string"
		}

		parsed = append(parsed, f)
	}

	return parsed
}

// Content generators
func (c *GenerateCommand) generateModelContent(name string, fields []Field) string {
	var fieldsStr string
	for _, f := range fields {
		tags := ""
		if contains(f.Modifiers, "unique") {
			tags += "unique"
		}
		if contains(f.Modifiers, "index") {
			if tags != "" {
				tags += ";"
			}
			tags += "index"
		}
		if tags != "" {
			tags = fmt.Sprintf(" `gorm:\"%s\"`", tags)
		}
		fieldsStr += fmt.Sprintf("\t%s %s%s\n", strings.Title(f.Name), f.GoType, tags)
	}

	return fmt.Sprintf(`package models

import (
	"time"
)

// %s model
type %s struct {
	ApplicationRecord
%s}

// TableName specifies the table name
func (%s) TableName() string {
	return "%ss"
}

// Validations
func (m *%s) Validate() error {
	// Add validations here
	return nil
}

// Callbacks
func (m *%s) BeforeCreate() error {
	return m.Validate()
}

func (m *%s) BeforeUpdate() error {
	return m.Validate()
}
`, name, name, fieldsStr, name, strings.ToLower(name), name, name, name)
}

func (c *GenerateCommand) generateFullControllerContent(name, modelName string) string {
	lowerName := strings.ToLower(modelName)
	pluralName := lowerName + "s"

	return fmt.Sprintf(`package controllers

import (
	"net/http"
	"github.com/cuemby/gor/pkg/gor"
	"../models"
)

// %s handles %s resources
type %s struct {
	ApplicationController
}

// Index displays all %s
func (c *%s) Index(ctx *gor.Context) error {
	var %s []models.%s
	
	if err := ctx.App().ORM().Find(&%s); err != nil {
		return err
	}

	return ctx.Render("%s/index", map[string]interface{}{
		"%s": %s,
	})
}

// Show displays a specific %s
func (c *%s) Show(ctx *gor.Context) error {
	var %s models.%s
	id := ctx.Param("id")

	if err := ctx.App().ORM().First(&%s, id); err != nil {
		return ctx.HTML(http.StatusNotFound, "Not Found")
	}

	return ctx.Render("%s/show", map[string]interface{}{
		"%s": %s,
	})
}

// New displays form for creating new %s
func (c *%s) New(ctx *gor.Context) error {
	%s := &models.%s{}
	return ctx.Render("%s/new", map[string]interface{}{
		"%s": %s,
	})
}

// Create processes creation of new %s
func (c *%s) Create(ctx *gor.Context) error {
	%s := &models.%s{}

	// Bind form data
	if err := ctx.Bind(%s); err != nil {
		return err
	}

	// Save to database
	if err := ctx.App().ORM().Create(%s); err != nil {
		return ctx.Render("%s/new", map[string]interface{}{
			"%s": %s,
			"error": err.Error(),
		})
	}

	return ctx.Redirect(http.StatusSeeOther, "/%s/"+fmt.Sprint(%s.ID))
}

// Edit displays form for editing %s
func (c *%s) Edit(ctx *gor.Context) error {
	var %s models.%s
	id := ctx.Param("id")

	if err := ctx.App().ORM().First(&%s, id); err != nil {
		return ctx.HTML(http.StatusNotFound, "Not Found")
	}

	return ctx.Render("%s/edit", map[string]interface{}{
		"%s": %s,
	})
}

// Update processes %s updates
func (c *%s) Update(ctx *gor.Context) error {
	var %s models.%s
	id := ctx.Param("id")

	if err := ctx.App().ORM().First(&%s, id); err != nil {
		return ctx.HTML(http.StatusNotFound, "Not Found")
	}

	// Bind form data
	if err := ctx.Bind(&%s); err != nil {
		return err
	}

	// Update in database
	if err := ctx.App().ORM().Update(&%s); err != nil {
		return ctx.Render("%s/edit", map[string]interface{}{
			"%s": %s,
			"error": err.Error(),
		})
	}

	return ctx.Redirect(http.StatusSeeOther, "/%s/"+fmt.Sprint(%s.ID))
}

// Destroy deletes a %s
func (c *%s) Destroy(ctx *gor.Context) error {
	id := ctx.Param("id")

	if err := ctx.App().ORM().Delete(&models.%s{}, id); err != nil {
		return err
	}

	return ctx.Redirect(http.StatusSeeOther, "/%s")
}
`,
		name, modelName, name,
		pluralName, name, pluralName, modelName, pluralName,
		pluralName, pluralName, pluralName,
		lowerName, name, lowerName, modelName, lowerName,
		pluralName, lowerName, lowerName,
		lowerName, name, lowerName, modelName, pluralName, lowerName, lowerName,
		lowerName, name, lowerName, modelName, lowerName, lowerName, pluralName, lowerName, lowerName,
		pluralName, lowerName,
		lowerName, name, lowerName, modelName, lowerName,
		pluralName, lowerName, lowerName,
		lowerName, name, lowerName, modelName, lowerName,
		lowerName, lowerName, pluralName, lowerName, lowerName,
		pluralName, lowerName,
		lowerName, name, modelName, pluralName)
}

func (c *GenerateCommand) generateControllerContent(name, modelName string, actions []string) string {
	// Simplified version - just generate specified actions
	var methods string
	for _, action := range actions {
		methods += fmt.Sprintf(`
// %s action
func (c *%s) %s(ctx *gor.Context) error {
	// TODO: Implement %s action
	return ctx.HTML(http.StatusOK, "%s action")
}
`, strings.Title(action), name, strings.Title(action), action, action)
	}

	return fmt.Sprintf(`package controllers

import (
	"net/http"
	"github.com/cuemby/gor/pkg/gor"
)

// %s controller
type %s struct {
	ApplicationController
}
%s
`, name, name, methods)
}

func (c *GenerateCommand) generateMigrationContent(name string, fields []string) string {
	if strings.HasPrefix(name, "create_") {
		return c.generateCreateTableMigration(name, fields)
	}
	return c.generateChangeMigration(name)
}

func (c *GenerateCommand) generateCreateTableMigration(name string, fields []string) string {
	tableName := strings.TrimPrefix(name, "create_")
	parsedFields := c.parseFields(fields)

	var fieldsSQL string
	for _, f := range parsedFields {
		sqlType := c.goTypeToSQL(f.Type)
		constraints := ""
		if contains(f.Modifiers, "unique") {
			constraints += " UNIQUE"
		}
		if contains(f.Modifiers, "required") {
			constraints += " NOT NULL"
		}
		fieldsSQL += fmt.Sprintf("\t\t%s %s%s,\n", f.Name, sqlType, constraints)
	}

	return fmt.Sprintf(`package migrations

import (
	"database/sql"
)

// Up migrates the database forward
func Up_%s(db *sql.DB) error {
	query := `+"`"+`
	CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
%s		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)
	`+"`"+`
	_, err := db.Exec(query)
	return err
}

// Down rolls back the migration
func Down_%s(db *sql.DB) error {
	_, err := db.Exec("DROP TABLE IF EXISTS %s")
	return err
}
`, name, tableName, fieldsSQL, name, tableName)
}

func (c *GenerateCommand) generateChangeMigration(name string) string {
	return fmt.Sprintf(`package migrations

import (
	"database/sql"
)

// Up migrates the database forward
func Up_%s(db *sql.DB) error {
	// TODO: Add your migration SQL here
	// Example: _, err := db.Exec("ALTER TABLE users ADD COLUMN age INTEGER")
	return nil
}

// Down rolls back the migration
func Down_%s(db *sql.DB) error {
	// TODO: Add your rollback SQL here
	// Example: _, err := db.Exec("ALTER TABLE users DROP COLUMN age")
	return nil
}
`, name, name)
}

func (c *GenerateCommand) generateViewContent(modelName, viewType string, fields []string) string {
	lowerName := strings.ToLower(modelName)
	pluralName := lowerName + "s"

	switch viewType {
	case "index":
		return fmt.Sprintf(`<h1>%s</h1>

<a href="/%s/new" class="btn btn-primary">New %s</a>

<table class="table">
  <thead>
    <tr>
      <th>ID</th>
      <th>Actions</th>
    </tr>
  </thead>
  <tbody>
    {{range .%s}}
    <tr>
      <td>{{.ID}}</td>
      <td>
        <a href="/%s/{{.ID}}">Show</a>
        <a href="/%s/{{.ID}}/edit">Edit</a>
        <form method="POST" action="/%s/{{.ID}}" style="display:inline">
          <input type="hidden" name="_method" value="DELETE">
          <button type="submit" onclick="return confirm('Are you sure?')">Delete</button>
        </form>
      </td>
    </tr>
    {{end}}
  </tbody>
</table>
`, strings.Title(pluralName), pluralName, modelName, pluralName, pluralName, pluralName, pluralName)

	case "show":
		return fmt.Sprintf(`<h1>%s #{{.%s.ID}}</h1>

<p>
  <a href="/%s/{{.%s.ID}}/edit">Edit</a>
  <a href="/%s">Back to List</a>
</p>
`, strings.Title(modelName), lowerName, pluralName, lowerName, pluralName)

	case "new":
		return fmt.Sprintf(`<h1>New %s</h1>

<form method="POST" action="/%s">
  {{template "form" .}}
  <button type="submit">Create %s</button>
  <a href="/%s">Cancel</a>
</form>
`, strings.Title(modelName), pluralName, modelName, pluralName)

	case "edit":
		return fmt.Sprintf(`<h1>Edit %s</h1>

<form method="POST" action="/%s/{{.%s.ID}}">
  <input type="hidden" name="_method" value="PUT">
  {{template "form" .}}
  <button type="submit">Update %s</button>
  <a href="/%s/{{.%s.ID}}">Cancel</a>
</form>
`, strings.Title(modelName), pluralName, lowerName, modelName, pluralName, lowerName)

	case "_form":
		var formFields string
		for _, field := range c.parseFields(fields) {
			inputType := "text"
			if field.Type == "text" {
				formFields += fmt.Sprintf(`  <div class="form-group">
    <label for="%s">%s</label>
    <textarea name="%s" id="%s">{{.%s.%s}}</textarea>
  </div>
`, field.Name, strings.Title(field.Name), field.Name, field.Name, lowerName, strings.Title(field.Name))
			} else {
				if field.Type == "boolean" {
					inputType = "checkbox"
				} else if field.Type == "integer" || field.Type == "float" {
					inputType = "number"
				} else if field.Type == "date" {
					inputType = "date"
				} else if field.Type == "datetime" {
					inputType = "datetime-local"
				}
				formFields += fmt.Sprintf(`  <div class="form-group">
    <label for="%s">%s</label>
    <input type="%s" name="%s" id="%s" value="{{.%s.%s}}">
  </div>
`, field.Name, strings.Title(field.Name), inputType, field.Name, field.Name, lowerName, strings.Title(field.Name))
			}
		}
		return fmt.Sprintf(`{{define "form"}}
%s{{end}}
`, formFields)

	default:
		return "<!-- View not implemented -->"
	}
}

func (c *GenerateCommand) generateJobContent(name string) string {
	return fmt.Sprintf(`package jobs

import (
	"context"
	"log"
)

// %s background job
type %s struct{}

// Perform executes the job
func (j *%s) Perform(ctx context.Context, args map[string]interface{}) error {
	log.Printf("%s job started with args: %%v", args)
	
	// TODO: Implement job logic here
	
	log.Printf("%s job completed")
	return nil
}
`, name, name, name, name, name)
}

func (c *GenerateCommand) generateMailerContent(name string) string {
	return fmt.Sprintf(`package mailers

import (
	"github.com/cuemby/gor/pkg/gor"
)

// %s mailer
type %s struct {
	gor.BaseMailer
}

// WelcomeEmail sends a welcome email
func (m *%s) WelcomeEmail(to string, data map[string]interface{}) error {
	return m.Mail(gor.MailOptions{
		To:      to,
		Subject: "Welcome!",
		Template: "mailers/welcome",
		Data:    data,
	})
}
`, name, name, name)
}

func (c *GenerateCommand) generateChannelContent(name string) string {
	return fmt.Sprintf(`package channels

import (
	"github.com/cuemby/gor/pkg/gor"
)

// %s WebSocket channel
type %s struct {
	gor.BaseChannel
}

// Subscribe handles new subscriptions
func (c *%s) Subscribe(ctx *gor.ChannelContext) error {
	// Handle subscription
	return ctx.Transmit("Welcome to %s channel!")
}

// Receive handles incoming messages
func (c *%s) Receive(ctx *gor.ChannelContext, message interface{}) error {
	// Handle incoming message
	return ctx.Broadcast(message)
}

// Unsubscribe handles disconnections
func (c *%s) Unsubscribe(ctx *gor.ChannelContext) error {
	// Handle unsubscription
	return nil
}
`, name, name, name, name, name, name)
}

func (c *GenerateCommand) goTypeToSQL(goType string) string {
	switch goType {
	case "string", "text":
		return "TEXT"
	case "integer":
		return "INTEGER"
	case "float":
		return "REAL"
	case "boolean":
		return "BOOLEAN"
	case "datetime":
		return "TIMESTAMP"
	case "date":
		return "DATE"
	case "time":
		return "TIME"
	default:
		return "TEXT"
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
