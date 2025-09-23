package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Deployer manages application deployment
type Deployer struct {
	config       *Config
	docker       *DockerBuilder
	orchestrator Orchestrator
	logger       Logger
}

// Config holds deployment configuration
type Config struct {
	AppName      string                 `json:"app_name"`
	Environment  string                 `json:"environment"`
	Version      string                 `json:"version"`
	Registry     string                 `json:"registry"`
	Servers      []Server               `json:"servers"`
	EnvVars      map[string]string      `json:"env_vars"`
	Secrets      map[string]string      `json:"secrets"`
	HealthCheck  HealthCheckConfig      `json:"health_check"`
	Rollback     RollbackConfig         `json:"rollback"`
	Deployment   DeploymentStrategy     `json:"deployment"`
	Database     DatabaseConfig         `json:"database"`
	SSL          SSLConfig              `json:"ssl"`
	Accessories  []AccessoryConfig      `json:"accessories"`
}

// Server represents a deployment target
type Server struct {
	Host     string            `json:"host"`
	User     string            `json:"user"`
	Port     int               `json:"port"`
	Roles    []string          `json:"roles"`
	Labels   map[string]string `json:"labels"`
	SSHKey   string            `json:"ssh_key"`
}

// HealthCheckConfig defines health check settings
type HealthCheckConfig struct {
	Path      string        `json:"path"`
	Interval  time.Duration `json:"interval"`
	Timeout   time.Duration `json:"timeout"`
	Retries   int           `json:"retries"`
}

// RollbackConfig defines rollback settings
type RollbackConfig struct {
	Enabled        bool `json:"enabled"`
	VersionsToKeep int  `json:"versions_to_keep"`
}

// DeploymentStrategy defines how to deploy
type DeploymentStrategy struct {
	Type            string        `json:"type"` // "rolling", "blue_green", "canary"
	MaxSurge        int           `json:"max_surge"`
	MaxUnavailable  int           `json:"max_unavailable"`
	CanaryPercent   int           `json:"canary_percent"`
	WaitTime        time.Duration `json:"wait_time"`
}

// DatabaseConfig defines database settings
type DatabaseConfig struct {
	URL            string `json:"url"`
	MigrateOnDeploy bool   `json:"migrate_on_deploy"`
	BackupBeforeMigrate bool `json:"backup_before_migrate"`
}

// SSLConfig defines SSL settings
type SSLConfig struct {
	Enabled     bool   `json:"enabled"`
	CertPath    string `json:"cert_path"`
	KeyPath     string `json:"key_path"`
	LetsEncrypt bool   `json:"lets_encrypt"`
	Domains     []string `json:"domains"`
}

// AccessoryConfig defines additional services
type AccessoryConfig struct {
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	Host    string            `json:"host"`
	Port    int               `json:"port"`
	Env     map[string]string `json:"env"`
	Volumes []string          `json:"volumes"`
	Cmd     []string          `json:"cmd"`
}

// Orchestrator defines the deployment orchestrator interface
type Orchestrator interface {
	Deploy(ctx context.Context, config *Config) error
	Rollback(ctx context.Context, config *Config, version string) error
	Scale(ctx context.Context, config *Config, replicas int) error
	Status(ctx context.Context, config *Config) (DeploymentStatus, error)
	Cleanup(ctx context.Context, config *Config) error
}

// DeploymentStatus represents deployment status
type DeploymentStatus struct {
	Version   string
	Replicas  int
	Available int
	Updated   int
	Ready     bool
	Message   string
}

// Logger defines the logging interface
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// NewDeployer creates a new deployer
func NewDeployer(config *Config, logger Logger) *Deployer {
	d := &Deployer{
		config: config,
		docker: NewDockerBuilder(config.AppName, config.Registry),
		logger: logger,
	}

	// Select orchestrator based on deployment type
	if len(config.Servers) == 1 {
		d.orchestrator = NewSSHOrchestrator(logger)
	} else {
		d.orchestrator = NewSwarmOrchestrator(logger)
	}

	return d
}

// Deploy deploys the application
func (d *Deployer) Deploy(ctx context.Context) error {
	d.logger.Info("Starting deployment for %s version %s", d.config.AppName, d.config.Version)

	// Step 1: Build and push Docker image
	if err := d.buildAndPush(ctx); err != nil {
		return fmt.Errorf("failed to build and push image: %w", err)
	}

	// Step 2: Run pre-deploy hooks
	if err := d.runPreDeployHooks(ctx); err != nil {
		return fmt.Errorf("pre-deploy hooks failed: %w", err)
	}

	// Step 3: Deploy using orchestrator
	if err := d.orchestrator.Deploy(ctx, d.config); err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	// Step 4: Run database migrations if needed
	if d.config.Database.MigrateOnDeploy {
		if err := d.runMigrations(ctx); err != nil {
			return fmt.Errorf("migrations failed: %w", err)
		}
	}

	// Step 5: Health check
	if err := d.waitForHealthy(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// Step 6: Run post-deploy hooks
	if err := d.runPostDeployHooks(ctx); err != nil {
		return fmt.Errorf("post-deploy hooks failed: %w", err)
	}

	// Step 7: Clean up old versions
	if d.config.Rollback.VersionsToKeep > 0 {
		if err := d.orchestrator.Cleanup(ctx, d.config); err != nil {
			d.logger.Error("Failed to clean up old versions: %v", err)
		}
	}

	d.logger.Info("Deployment completed successfully")
	return nil
}

// Rollback rolls back to a previous version
func (d *Deployer) Rollback(ctx context.Context, version string) error {
	if !d.config.Rollback.Enabled {
		return fmt.Errorf("rollback is not enabled")
	}

	d.logger.Info("Rolling back to version %s", version)

	if err := d.orchestrator.Rollback(ctx, d.config, version); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	if err := d.waitForHealthy(ctx); err != nil {
		return fmt.Errorf("health check after rollback failed: %w", err)
	}

	d.logger.Info("Rollback completed successfully")
	return nil
}

// Scale scales the deployment
func (d *Deployer) Scale(ctx context.Context, replicas int) error {
	d.logger.Info("Scaling to %d replicas", replicas)

	if err := d.orchestrator.Scale(ctx, d.config, replicas); err != nil {
		return fmt.Errorf("scaling failed: %w", err)
	}

	d.logger.Info("Scaling completed successfully")
	return nil
}

// Status gets deployment status
func (d *Deployer) Status(ctx context.Context) (DeploymentStatus, error) {
	return d.orchestrator.Status(ctx, d.config)
}

// buildAndPush builds and pushes Docker image
func (d *Deployer) buildAndPush(ctx context.Context) error {
	d.logger.Info("Building Docker image")

	// Build image
	if err := d.docker.Build(ctx, ".", d.config.Version); err != nil {
		return err
	}

	// Push to registry
	if d.config.Registry != "" {
		d.logger.Info("Pushing image to registry")
		if err := d.docker.Push(ctx, d.config.Version); err != nil {
			return err
		}
	}

	return nil
}

// runPreDeployHooks runs pre-deployment hooks
func (d *Deployer) runPreDeployHooks(ctx context.Context) error {
	hookPath := filepath.Join(".gor", "hooks", "pre-deploy")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		return nil
	}

	d.logger.Info("Running pre-deploy hooks")
	cmd := exec.CommandContext(ctx, hookPath)
	cmd.Env = d.buildEnv()
	return cmd.Run()
}

// runPostDeployHooks runs post-deployment hooks
func (d *Deployer) runPostDeployHooks(ctx context.Context) error {
	hookPath := filepath.Join(".gor", "hooks", "post-deploy")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		return nil
	}

	d.logger.Info("Running post-deploy hooks")
	cmd := exec.CommandContext(ctx, hookPath)
	cmd.Env = d.buildEnv()
	return cmd.Run()
}

// runMigrations runs database migrations
func (d *Deployer) runMigrations(ctx context.Context) error {
	d.logger.Info("Running database migrations")

	if d.config.Database.BackupBeforeMigrate {
		if err := d.backupDatabase(ctx); err != nil {
			return fmt.Errorf("database backup failed: %w", err)
		}
	}

	// Run migrations on one of the servers
	for _, server := range d.config.Servers {
		for _, role := range server.Roles {
			if role == "db" || role == "app" {
				return d.runMigrationOnServer(ctx, server)
			}
		}
	}

	return nil
}

// runMigrationOnServer runs migrations on a specific server
func (d *Deployer) runMigrationOnServer(ctx context.Context, server Server) error {
	cmd := fmt.Sprintf("docker run --rm -e DATABASE_URL=%s %s/%s:%s gor migrate",
		d.config.Database.URL,
		d.config.Registry,
		d.config.AppName,
		d.config.Version,
	)

	return d.executeSSH(ctx, server, cmd)
}

// backupDatabase backs up the database
func (d *Deployer) backupDatabase(ctx context.Context) error {
	d.logger.Info("Backing up database")
	// Implementation would depend on database type
	return nil
}

// waitForHealthy waits for deployment to be healthy
func (d *Deployer) waitForHealthy(ctx context.Context) error {
	d.logger.Info("Waiting for deployment to be healthy")

	start := time.Now()
	timeout := 5 * time.Minute

	for {
		status, err := d.orchestrator.Status(ctx, d.config)
		if err != nil {
			return err
		}

		if status.Ready {
			d.logger.Info("Deployment is healthy")
			return nil
		}

		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for deployment to be healthy")
		}

		time.Sleep(5 * time.Second)
	}
}

// executeSSH executes a command on a server via SSH
func (d *Deployer) executeSSH(ctx context.Context, server Server, command string) error {
	sshCmd := fmt.Sprintf("ssh -i %s -p %d %s@%s %s",
		server.SSHKey,
		server.Port,
		server.User,
		server.Host,
		command,
	)

	cmd := exec.CommandContext(ctx, "sh", "-c", sshCmd)
	return cmd.Run()
}

// buildEnv builds environment variables
func (d *Deployer) buildEnv() []string {
	env := os.Environ()

	for key, value := range d.config.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	env = append(env, fmt.Sprintf("GOR_ENV=%s", d.config.Environment))
	env = append(env, fmt.Sprintf("GOR_VERSION=%s", d.config.Version))

	return env
}