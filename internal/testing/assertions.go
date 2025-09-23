package testing

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Assertions provides test assertion methods
type Assertions struct {
	t *testing.T
}

// NewAssertions creates a new assertions helper
func NewAssertions(t *testing.T) *Assertions {
	return &Assertions{t: t}
}

// Equal asserts that two values are equal
func (a *Assertions) Equal(expected, actual interface{}, msgAndArgs ...interface{}) bool {
	if !reflect.DeepEqual(expected, actual) {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected %v, got %v. %s", expected, actual, msg)
		return false
	}
	return true
}

// NotEqual asserts that two values are not equal
func (a *Assertions) NotEqual(expected, actual interface{}, msgAndArgs ...interface{}) bool {
	if reflect.DeepEqual(expected, actual) {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected values to be different, both are %v. %s", actual, msg)
		return false
	}
	return true
}

// Nil asserts that a value is nil
func (a *Assertions) Nil(value interface{}, msgAndArgs ...interface{}) bool {
	if !isNil(value) {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected nil, got %v. %s", value, msg)
		return false
	}
	return true
}

// NotNil asserts that a value is not nil
func (a *Assertions) NotNil(value interface{}, msgAndArgs ...interface{}) bool {
	if isNil(value) {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected non-nil value. %s", msg)
		return false
	}
	return true
}

// True asserts that a value is true
func (a *Assertions) True(value bool, msgAndArgs ...interface{}) bool {
	if !value {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected true, got false. %s", msg)
		return false
	}
	return true
}

// False asserts that a value is false
func (a *Assertions) False(value bool, msgAndArgs ...interface{}) bool {
	if value {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected false, got true. %s", msg)
		return false
	}
	return true
}

// Contains asserts that a string contains a substring
func (a *Assertions) Contains(s, substr string, msgAndArgs ...interface{}) bool {
	if !strings.Contains(s, substr) {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected %q to contain %q. %s", s, substr, msg)
		return false
	}
	return true
}

// NotContains asserts that a string does not contain a substring
func (a *Assertions) NotContains(s, substr string, msgAndArgs ...interface{}) bool {
	if strings.Contains(s, substr) {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected %q to not contain %q. %s", s, substr, msg)
		return false
	}
	return true
}

// Empty asserts that a value is empty
func (a *Assertions) Empty(value interface{}, msgAndArgs ...interface{}) bool {
	if !isEmpty(value) {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected empty value, got %v. %s", value, msg)
		return false
	}
	return true
}

// NotEmpty asserts that a value is not empty
func (a *Assertions) NotEmpty(value interface{}, msgAndArgs ...interface{}) bool {
	if isEmpty(value) {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected non-empty value. %s", msg)
		return false
	}
	return true
}

// Len asserts that a value has a specific length
func (a *Assertions) Len(value interface{}, length int, msgAndArgs ...interface{}) bool {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var actualLen int
	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
		actualLen = v.Len()
	default:
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Cannot get length of %v. %s", value, msg)
		return false
	}

	if actualLen != length {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected length %d, got %d. %s", length, actualLen, msg)
		return false
	}
	return true
}

// Error asserts that an error occurred
func (a *Assertions) Error(err error, msgAndArgs ...interface{}) bool {
	if err == nil {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected error, got nil. %s", msg)
		return false
	}
	return true
}

// NoError asserts that no error occurred
func (a *Assertions) NoError(err error, msgAndArgs ...interface{}) bool {
	if err != nil {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected no error, got %v. %s", err, msg)
		return false
	}
	return true
}

// Panics asserts that a function panics
func (a *Assertions) Panics(fn func(), msgAndArgs ...interface{}) bool {
	defer func() {
		if r := recover(); r == nil {
			msg := formatMessage(msgAndArgs...)
			a.t.Errorf("Expected panic, but function completed normally. %s", msg)
		}
	}()

	fn()
	return true
}

// NotPanics asserts that a function does not panic
func (a *Assertions) NotPanics(fn func(), msgAndArgs ...interface{}) bool {
	defer func() {
		if r := recover(); r != nil {
			msg := formatMessage(msgAndArgs...)
			a.t.Errorf("Expected no panic, but got %v. %s", r, msg)
		}
	}()

	fn()
	return true
}

// HTTPSuccess asserts that an HTTP status code is successful (2xx)
func (a *Assertions) HTTPSuccess(statusCode int, msgAndArgs ...interface{}) bool {
	if statusCode < 200 || statusCode >= 300 {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected successful status code (2xx), got %d. %s", statusCode, msg)
		return false
	}
	return true
}

// HTTPError asserts that an HTTP status code is an error (4xx or 5xx)
func (a *Assertions) HTTPError(statusCode int, msgAndArgs ...interface{}) bool {
	if statusCode < 400 {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected error status code (4xx or 5xx), got %d. %s", statusCode, msg)
		return false
	}
	return true
}

// HTTPStatusCode asserts a specific HTTP status code
func (a *Assertions) HTTPStatusCode(expected, actual int, msgAndArgs ...interface{}) bool {
	if expected != actual {
		msg := formatMessage(msgAndArgs...)
		a.t.Errorf("Expected status code %d, got %d. %s", expected, actual, msg)
		return false
	}
	return true
}

// Helper functions

func isNil(value interface{}) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func:
		return v.IsNil()
	}
	return false
}

func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
		return v.Len() == 0
	case reflect.Ptr:
		if v.IsNil() {
			return true
		}
		return isEmpty(v.Elem().Interface())
	}
	return false
}

func formatMessage(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 {
		return ""
	}

	if len(msgAndArgs) == 1 {
		return fmt.Sprintf("%v", msgAndArgs[0])
	}

	if format, ok := msgAndArgs[0].(string); ok {
		return fmt.Sprintf(format, msgAndArgs[1:]...)
	}

	return fmt.Sprint(msgAndArgs...)
}
