package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Environment represents application environment configuration
type Environment struct {
	Name     string                 `yaml:"name" json:"name"`
	Settings map[string]interface{} `yaml:"settings" json:"settings"`
}

// Environments manages multiple environment configurations
type Environments struct {
	environments map[string]*Environment
	current      string
	basePath     string
}

// NewEnvironments creates a new environments manager
func NewEnvironments(basePath string) *Environments {
	return &Environments{
		environments: make(map[string]*Environment),
		current:      getEnvironment(),
		basePath:     basePath,
	}
}

// Load loads all environment configurations
func (e *Environments) Load() error {
	// Define standard environments
	standardEnvs := []string{"development", "test", "staging", "production"}

	for _, env := range standardEnvs {
		envFile := filepath.Join(e.basePath, fmt.Sprintf("config.%s.yml", env))
		if _, err := os.Stat(envFile); err == nil {
			if err := e.loadEnvironment(env, envFile); err != nil {
				return err
			}
		}
	}

	// Load custom environments
	pattern := filepath.Join(e.basePath, "config.*.yml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, match := range matches {
		base := filepath.Base(match)
		// Extract environment name from filename
		if len(base) > 11 && base[:7] == "config." && base[len(base)-4:] == ".yml" {
			envName := base[7 : len(base)-4]
			// Skip if already loaded
			if _, exists := e.environments[envName]; !exists {
				if err := e.loadEnvironment(envName, match); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// loadEnvironment loads a specific environment configuration
func (e *Environments) loadEnvironment(name, path string) error {
	config, err := New(e.basePath)
	if err != nil {
		return err
	}
	if err := config.loadFile(path); err != nil {
		return err
	}

	e.environments[name] = &Environment{
		Name:     name,
		Settings: config.data,
	}

	return nil
}

// Get gets configuration for a specific environment
func (e *Environments) Get(name string) (*Environment, error) {
	if env, ok := e.environments[name]; ok {
		return env, nil
	}
	return nil, fmt.Errorf("environment %s not found", name)
}

// Current gets the current environment
func (e *Environments) Current() *Environment {
	if env, ok := e.environments[e.current]; ok {
		return env
	}
	return nil
}

// SetCurrent sets the current environment
func (e *Environments) SetCurrent(name string) error {
	if _, ok := e.environments[name]; !ok {
		return fmt.Errorf("environment %s not found", name)
	}
	e.current = name
	return nil
}

// List lists all available environments
func (e *Environments) List() []string {
	envs := make([]string, 0, len(e.environments))
	for name := range e.environments {
		envs = append(envs, name)
	}
	return envs
}

// DefaultEnvironmentConfig creates default environment configurations
func DefaultEnvironmentConfig() map[string]interface{} {
	return map[string]interface{}{
		"development": map[string]interface{}{
			"app": map[string]interface{}{
				"name":  "Gor Application",
				"debug": true,
				"port":  3000,
				"host":  "localhost",
			},
			"database": map[string]interface{}{
				"driver":   "sqlite3",
				"database": "development.db",
				"pool":     5,
				"log":      true,
			},
			"cache": map[string]interface{}{
				"driver":     "memory",
				"ttl":        3600,
				"max_items":  1000,
			},
			"queue": map[string]interface{}{
				"driver":  "database",
				"workers": 2,
				"retries": 3,
			},
			"session": map[string]interface{}{
				"driver":     "cookie",
				"secure":     false,
				"http_only":  true,
				"lifetime":   86400,
				"same_site":  "lax",
			},
			"log": map[string]interface{}{
				"level":  "debug",
				"format": "text",
				"output": "stdout",
			},
			"assets": map[string]interface{}{
				"compile":     true,
				"minify":      false,
				"fingerprint": false,
				"watch":       true,
			},
		},
		"test": map[string]interface{}{
			"app": map[string]interface{}{
				"name":  "Gor Test",
				"debug": false,
				"port":  3001,
				"host":  "localhost",
			},
			"database": map[string]interface{}{
				"driver":   "sqlite3",
				"database": ":memory:",
				"pool":     1,
				"log":      false,
			},
			"cache": map[string]interface{}{
				"driver": "memory",
				"ttl":    60,
			},
			"queue": map[string]interface{}{
				"driver":  "memory",
				"workers": 1,
			},
			"session": map[string]interface{}{
				"driver": "memory",
			},
			"log": map[string]interface{}{
				"level":  "error",
				"format": "json",
				"output": "stdout",
			},
		},
		"staging": map[string]interface{}{
			"app": map[string]interface{}{
				"name":  "Gor Staging",
				"debug": false,
				"port":  3000,
				"host":  "0.0.0.0",
			},
			"database": map[string]interface{}{
				"driver":   "postgres",
				"host":     "${DB_HOST}",
				"port":     5432,
				"database": "${DB_NAME}",
				"user":     "${DB_USER}",
				"password": "${DB_PASSWORD}",
				"pool":     10,
				"log":      false,
			},
			"cache": map[string]interface{}{
				"driver":   "redis",
				"host":     "${REDIS_HOST}",
				"port":     6379,
				"password": "${REDIS_PASSWORD}",
				"database": 0,
				"ttl":      7200,
			},
			"queue": map[string]interface{}{
				"driver":  "redis",
				"workers": 5,
				"retries": 3,
			},
			"session": map[string]interface{}{
				"driver":     "redis",
				"secure":     true,
				"http_only":  true,
				"lifetime":   86400,
				"same_site":  "strict",
			},
			"log": map[string]interface{}{
				"level":  "info",
				"format": "json",
				"output": "file",
				"path":   "/var/log/gor/staging.log",
			},
			"assets": map[string]interface{}{
				"compile":     false,
				"minify":      true,
				"fingerprint": true,
				"cdn_url":     "${CDN_URL}",
			},
		},
		"production": map[string]interface{}{
			"app": map[string]interface{}{
				"name":  "Gor Production",
				"debug": false,
				"port":  3000,
				"host":  "0.0.0.0",
			},
			"database": map[string]interface{}{
				"driver":          "postgres",
				"host":            "${DB_HOST}",
				"port":            5432,
				"database":        "${DB_NAME}",
				"user":            "${DB_USER}",
				"password":        "${DB_PASSWORD}",
				"pool":            20,
				"max_idle":        5,
				"max_lifetime":    3600,
				"log":             false,
				"ssl_mode":        "require",
			},
			"cache": map[string]interface{}{
				"driver":   "redis",
				"host":     "${REDIS_HOST}",
				"port":     6379,
				"password": "${REDIS_PASSWORD}",
				"database": 0,
				"cluster":  true,
				"ttl":      14400,
			},
			"queue": map[string]interface{}{
				"driver":       "redis",
				"workers":      10,
				"retries":      5,
				"retry_delay":  60,
				"max_jobs":     1000,
			},
			"session": map[string]interface{}{
				"driver":     "redis",
				"secure":     true,
				"http_only":  true,
				"lifetime":   86400,
				"same_site":  "strict",
				"domain":     "${SESSION_DOMAIN}",
			},
			"log": map[string]interface{}{
				"level":     "warning",
				"format":    "json",
				"output":    "file",
				"path":      "/var/log/gor/production.log",
				"rotate":    true,
				"max_size":  100, // MB
				"max_files": 10,
			},
			"assets": map[string]interface{}{
				"compile":     false,
				"minify":      true,
				"fingerprint": true,
				"cdn_url":     "${CDN_URL}",
				"cache_control": "public, max-age=31536000",
			},
			"security": map[string]interface{}{
				"cors": map[string]interface{}{
					"enabled":      true,
					"origins":      []string{"${ALLOWED_ORIGINS}"},
					"credentials":  true,
					"max_age":      86400,
				},
				"csrf": map[string]interface{}{
					"enabled":     true,
					"token_length": 32,
				},
				"rate_limit": map[string]interface{}{
					"enabled":  true,
					"requests": 100,
					"window":   60, // seconds
				},
			},
		},
	}
}