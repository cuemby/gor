package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CLI provides deployment CLI commands
type CLI struct {
	config   *Config
	deployer *Deployer
	logger   Logger
}

// NewCLI creates a new deployment CLI
func NewCLI(configPath string, logger Logger) (*CLI, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &CLI{
		config:   config,
		deployer: NewDeployer(config, logger),
		logger:   logger,
	}, nil
}

// Deploy runs the deployment
func (c *CLI) Deploy() error {
	ctx := context.Background()
	return c.deployer.Deploy(ctx)
}

// Rollback rolls back to a previous version
func (c *CLI) Rollback(version string) error {
	ctx := context.Background()
	return c.deployer.Rollback(ctx, version)
}

// Scale scales the deployment
func (c *CLI) Scale(replicas int) error {
	ctx := context.Background()
	return c.deployer.Scale(ctx, replicas)
}

// Status shows deployment status
func (c *CLI) Status() error {
	ctx := context.Background()
	status, err := c.deployer.Status(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("Deployment Status:\n")
	fmt.Printf("  Version:   %s\n", status.Version)
	fmt.Printf("  Replicas:  %d\n", status.Replicas)
	fmt.Printf("  Available: %d\n", status.Available)
	fmt.Printf("  Ready:     %v\n", status.Ready)
	fmt.Printf("  Message:   %s\n", status.Message)

	return nil
}

// Setup initializes deployment configuration
func (c *CLI) Setup() error {
	// Create deploy directory
	deployDir := ".gor/deploy"
	if err := os.MkdirAll(deployDir, 0755); err != nil {
		return fmt.Errorf("failed to create deploy directory: %w", err)
	}

	// Generate default config
	defaultConfig := &Config{
		AppName:     "myapp",
		Environment: "production",
		Version:     "latest",
		Registry:    "docker.io/myregistry",
		Servers: []Server{
			{
				Host:   "server1.example.com",
				User:   "deploy",
				Port:   22,
				Roles:  []string{"app", "db"},
				SSHKey: "~/.ssh/id_rsa",
			},
		},
		EnvVars: map[string]string{
			"GOR_ENV":   "production",
			"LOG_LEVEL": "info",
			"PORT":      "3000",
		},
		Secrets: map[string]string{
			"SECRET_KEY_BASE": "${SECRET_KEY_BASE}",
			"DATABASE_URL":    "${DATABASE_URL}",
		},
		HealthCheck: HealthCheckConfig{
			Path:     "/health",
			Interval: 30 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
		Rollback: RollbackConfig{
			Enabled:        true,
			VersionsToKeep: 5,
		},
		Deployment: DeploymentStrategy{
			Type:           "rolling",
			MaxSurge:       1,
			MaxUnavailable: 0,
			WaitTime:       10 * time.Second,
		},
		Database: DatabaseConfig{
			URL:                 "${DATABASE_URL}",
			MigrateOnDeploy:     true,
			BackupBeforeMigrate: true,
		},
		SSL: SSLConfig{
			Enabled:     false,
			LetsEncrypt: true,
			Domains:     []string{"example.com", "www.example.com"},
		},
		Accessories: []AccessoryConfig{
			{
				Name:  "redis",
				Image: "redis:7-alpine",
				Port:  6379,
				Volumes: []string{
					"redis-data:/data",
				},
			},
		},
	}

	// Save config
	configPath := filepath.Join(deployDir, "config.json")
	if err := SaveConfig(configPath, defaultConfig); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Generate Docker Compose file
	if err := GenerateDockerCompose(defaultConfig); err != nil {
		return fmt.Errorf("failed to generate docker-compose: %w", err)
	}

	// Create hooks directory
	hooksDir := filepath.Join(deployDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Create example hooks
	preDeployHook := `#!/bin/bash
# Pre-deploy hook
echo "Running pre-deploy tasks..."

# Example: Run tests
# go test ./...

# Example: Build assets
# gor assets:compile

echo "Pre-deploy tasks completed"
`
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-deploy"), []byte(preDeployHook), 0755); err != nil {
		return fmt.Errorf("failed to create pre-deploy hook: %w", err)
	}

	postDeployHook := `#!/bin/bash
# Post-deploy hook
echo "Running post-deploy tasks..."

# Example: Clear cache
# gor cache:clear

# Example: Send notification
# curl -X POST https://hooks.slack.com/... -d '{"text":"Deployment completed"}'

echo "Post-deploy tasks completed"
`
	if err := os.WriteFile(filepath.Join(hooksDir, "post-deploy"), []byte(postDeployHook), 0755); err != nil {
		return fmt.Errorf("failed to create post-deploy hook: %w", err)
	}

	// Create environment file template
	envTemplate := `# Production Environment Variables
# Copy this file to .env.production and fill in the values

# Database
DATABASE_URL=postgres://user:pass@localhost/myapp_production

# Redis
REDIS_URL=redis://localhost:6379

# Secrets
SECRET_KEY_BASE=your-secret-key-here

# External Services
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=
AWS_REGION=us-east-1

# Mail
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=
SMTP_PASSWORD=

# Monitoring
SENTRY_DSN=
NEW_RELIC_LICENSE_KEY=
`
	if err := os.WriteFile(".env.production.example", []byte(envTemplate), 0644); err != nil {
		return fmt.Errorf("failed to create env template: %w", err)
	}

	fmt.Println("Deployment setup completed successfully!")
	fmt.Printf("Configuration saved to: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Edit .gor/deploy/config.json with your server details")
	fmt.Println("2. Copy .env.production.example to .env.production and fill in values")
	fmt.Println("3. Run 'gor deploy' to deploy your application")

	return nil
}

// LoadConfig loads deployment configuration
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Load environment-specific overrides
	envFile := fmt.Sprintf(".env.%s", config.Environment)
	if _, err := os.Stat(envFile); err == nil {
		if err := loadEnvFile(envFile, &config); err != nil {
			return nil, err
		}
	}

	// Set version from environment or git
	if config.Version == "" || config.Version == "latest" {
		config.Version = getVersion()
	}

	return &config, nil
}

// SaveConfig saves deployment configuration
func SaveConfig(path string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// loadEnvFile loads environment variables from a file
func loadEnvFile(path string, config *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Parse env file and update config
	// Simple implementation - in production use a proper env parser
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Update secrets if it's a secret key
			if strings.HasSuffix(key, "_KEY") || strings.HasSuffix(key, "_SECRET") ||
				strings.HasSuffix(key, "_PASSWORD") || key == "DATABASE_URL" {
				config.Secrets[key] = value
			} else {
				config.EnvVars[key] = value
			}
		}
	}

	return nil
}

// getVersion gets the current version from git or timestamp
func getVersion() string {
	// Try to get git commit hash
	if output, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output(); err == nil {
		return strings.TrimSpace(string(output))
	}

	// Fallback to timestamp
	return time.Now().Format("20060102150405")
}

// SimpleLogger provides basic logging
type SimpleLogger struct{}

func (l *SimpleLogger) Info(msg string, args ...interface{}) {
	fmt.Printf("[INFO] "+msg+"\n", args...)
}

func (l *SimpleLogger) Error(msg string, args ...interface{}) {
	fmt.Printf("[ERROR] "+msg+"\n", args...)
}

func (l *SimpleLogger) Debug(msg string, args ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		fmt.Printf("[DEBUG] "+msg+"\n", args...)
	}
}
