package config

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strings"
)

// Validator validates configuration values
type Validator struct {
	rules map[string][]ValidationRule
}

// ValidationRule defines a validation rule
type ValidationRule struct {
	Name      string
	Validator func(interface{}) error
	Message   string
}

// ValidationError represents a validation error
type ValidationError struct {
	Key     string
	Rule    string
	Message string
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

// Error returns the error message
func (ve ValidationErrors) Error() string {
	var messages []string
	for _, e := range ve {
		messages = append(messages, fmt.Sprintf("%s: %s", e.Key, e.Message))
	}
	return strings.Join(messages, "; ")
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		rules: make(map[string][]ValidationRule),
	}
}

// AddRule adds a validation rule for a key
func (v *Validator) AddRule(key string, rule ValidationRule) {
	v.rules[key] = append(v.rules[key], rule)
}

// Required adds a required validation rule
func (v *Validator) Required(key string) {
	v.AddRule(key, ValidationRule{
		Name: "required",
		Validator: func(value interface{}) error {
			if value == nil || value == "" || value == 0 {
				return fmt.Errorf("is required")
			}
			return nil
		},
		Message: "is required",
	})
}

// Min adds a minimum value validation rule
func (v *Validator) Min(key string, min float64) {
	v.AddRule(key, ValidationRule{
		Name: "min",
		Validator: func(value interface{}) error {
			switch v := value.(type) {
			case int:
				if float64(v) < min {
					return fmt.Errorf("must be at least %v", min)
				}
			case float64:
				if v < min {
					return fmt.Errorf("must be at least %v", min)
				}
			case string:
				if float64(len(v)) < min {
					return fmt.Errorf("must be at least %v characters", min)
				}
			}
			return nil
		},
		Message: fmt.Sprintf("must be at least %v", min),
	})
}

// Max adds a maximum value validation rule
func (v *Validator) Max(key string, max float64) {
	v.AddRule(key, ValidationRule{
		Name: "max",
		Validator: func(value interface{}) error {
			switch v := value.(type) {
			case int:
				if float64(v) > max {
					return fmt.Errorf("must be at most %v", max)
				}
			case float64:
				if v > max {
					return fmt.Errorf("must be at most %v", max)
				}
			case string:
				if float64(len(v)) > max {
					return fmt.Errorf("must be at most %v characters", max)
				}
			}
			return nil
		},
		Message: fmt.Sprintf("must be at most %v", max),
	})
}

// In adds an inclusion validation rule
func (v *Validator) In(key string, values []interface{}) {
	v.AddRule(key, ValidationRule{
		Name: "in",
		Validator: func(value interface{}) error {
			for _, allowed := range values {
				if reflect.DeepEqual(value, allowed) {
					return nil
				}
			}
			return fmt.Errorf("must be one of %v", values)
		},
		Message: fmt.Sprintf("must be one of %v", values),
	})
}

// Pattern adds a pattern validation rule
func (v *Validator) Pattern(key string, pattern string) {
	regex := regexp.MustCompile(pattern)
	v.AddRule(key, ValidationRule{
		Name: "pattern",
		Validator: func(value interface{}) error {
			str := fmt.Sprintf("%v", value)
			if !regex.MatchString(str) {
				return fmt.Errorf("must match pattern %s", pattern)
			}
			return nil
		},
		Message: fmt.Sprintf("must match pattern %s", pattern),
	})
}

// URL adds a URL validation rule
func (v *Validator) URL(key string) {
	v.AddRule(key, ValidationRule{
		Name: "url",
		Validator: func(value interface{}) error {
			str := fmt.Sprintf("%v", value)
			_, err := url.Parse(str)
			if err != nil {
				return fmt.Errorf("must be a valid URL")
			}
			return nil
		},
		Message: "must be a valid URL",
	})
}

// Email adds an email validation rule
func (v *Validator) Email(key string) {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	v.AddRule(key, ValidationRule{
		Name: "email",
		Validator: func(value interface{}) error {
			str := fmt.Sprintf("%v", value)
			if !emailRegex.MatchString(str) {
				return fmt.Errorf("must be a valid email address")
			}
			return nil
		},
		Message: "must be a valid email address",
	})
}

// Custom adds a custom validation rule
func (v *Validator) Custom(key string, validator func(interface{}) error, message string) {
	v.AddRule(key, ValidationRule{
		Name:      "custom",
		Validator: validator,
		Message:   message,
	})
}

// Validate validates a configuration
func (v *Validator) Validate(config *Config) ValidationErrors {
	var errors ValidationErrors

	for key, rules := range v.rules {
		value := config.Get(key)

		for _, rule := range rules {
			if err := rule.Validator(value); err != nil {
				errors = append(errors, ValidationError{
					Key:     key,
					Rule:    rule.Name,
					Message: err.Error(),
				})
			}
		}
	}

	if len(errors) == 0 {
		return nil
	}

	return errors
}

// DefaultValidation creates default validation rules
func DefaultValidation() *Validator {
	v := NewValidator()

	// App configuration
	v.Required("app.name")
	v.Required("app.port")
	v.Min("app.port", 1)
	v.Max("app.port", 65535)

	// Database configuration
	v.Required("database.driver")
	v.In("database.driver", []interface{}{"sqlite3", "postgres", "mysql"})

	// When using postgres or mysql, additional fields are required
	v.Custom("database", func(value interface{}) error {
		if m, ok := value.(map[string]interface{}); ok {
			driver := m["driver"]
			if driver == "postgres" || driver == "mysql" {
				if m["host"] == nil || m["host"] == "" {
					return fmt.Errorf("host is required for %s", driver)
				}
				if m["database"] == nil || m["database"] == "" {
					return fmt.Errorf("database name is required for %s", driver)
				}
			}
		}
		return nil
	}, "invalid database configuration")

	// Cache configuration
	v.Required("cache.driver")
	v.In("cache.driver", []interface{}{"memory", "redis", "database"})

	// Queue configuration
	v.Required("queue.driver")
	v.In("queue.driver", []interface{}{"memory", "database", "redis"})
	v.Min("queue.workers", 1)
	v.Max("queue.workers", 100)

	// Session configuration
	v.Required("session.driver")
	v.In("session.driver", []interface{}{"cookie", "memory", "redis", "database"})
	v.Min("session.lifetime", 60)

	// Log configuration
	v.Required("log.level")
	v.In("log.level", []interface{}{"debug", "info", "warning", "error", "fatal"})
	v.Required("log.format")
	v.In("log.format", []interface{}{"text", "json"})

	return v
}

// ValidateEnvironment validates environment-specific configuration
func ValidateEnvironment(env string, config *Config) error {
	v := NewValidator()

	switch env {
	case "production":
		// Production-specific validation
		v.Custom("app.debug", func(value interface{}) error {
			if b, ok := value.(bool); ok && b {
				return fmt.Errorf("debug mode should be disabled in production")
			}
			return nil
		}, "debug mode should be disabled in production")

		v.Custom("session.secure", func(value interface{}) error {
			if b, ok := value.(bool); ok && !b {
				return fmt.Errorf("secure cookies should be enabled in production")
			}
			return nil
		}, "secure cookies should be enabled in production")

		v.Custom("log.level", func(value interface{}) error {
			if level, ok := value.(string); ok && (level == "debug" || level == "info") {
				return fmt.Errorf("log level should be warning or higher in production")
			}
			return nil
		}, "log level should be warning or higher in production")

	case "test":
		// Test-specific validation
		v.Custom("database.driver", func(value interface{}) error {
			if driver, ok := value.(string); ok && driver != "sqlite3" {
				return fmt.Errorf("test environment should use sqlite3")
			}
			return nil
		}, "test environment should use sqlite3")
	}

	errors := v.Validate(config)
	if errors != nil {
		return errors
	}

	return nil
}
