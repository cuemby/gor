package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cuemby/gor/pkg/gor"
)

// Helper function to create a test context
func createTestContext(method, path string) (*gor.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)

	ctx := &gor.Context{
		Context:  context.Background(),
		Request:  req,
		Response: w,
		Params:   make(map[string]string),
		Query:    make(map[string][]string),
	}

	return ctx, w
}

func TestBaseControllerActions(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		path          string
		action        func(*BaseController, *gor.Context) error
		expectedError string
		setupParams   func(*gor.Context)
	}{
		{
			name:          "Index",
			method:        "GET",
			path:          "/",
			action:        func(c *BaseController, ctx *gor.Context) error { return c.Index(ctx) },
			expectedError: "Index action not implemented",
		},
		{
			name:          "Show",
			method:        "GET",
			path:          "/123",
			action:        func(c *BaseController, ctx *gor.Context) error { return c.Show(ctx) },
			expectedError: "Show action not implemented",
			setupParams: func(ctx *gor.Context) {
				ctx.Params["id"] = "123"
			},
		},
		{
			name:          "New",
			method:        "GET",
			path:          "/new",
			action:        func(c *BaseController, ctx *gor.Context) error { return c.New(ctx) },
			expectedError: "New action not implemented",
		},
		{
			name:          "Create",
			method:        "POST",
			path:          "/",
			action:        func(c *BaseController, ctx *gor.Context) error { return c.Create(ctx) },
			expectedError: "Create action not implemented",
		},
		{
			name:          "Edit",
			method:        "GET",
			path:          "/456/edit",
			action:        func(c *BaseController, ctx *gor.Context) error { return c.Edit(ctx) },
			expectedError: "Edit action not implemented",
			setupParams: func(ctx *gor.Context) {
				ctx.Params["id"] = "456"
			},
		},
		{
			name:          "Update",
			method:        "PUT",
			path:          "/789",
			action:        func(c *BaseController, ctx *gor.Context) error { return c.Update(ctx) },
			expectedError: "Update action not implemented",
			setupParams: func(ctx *gor.Context) {
				ctx.Params["id"] = "789"
			},
		},
		{
			name:          "Destroy",
			method:        "DELETE",
			path:          "/999",
			action:        func(c *BaseController, ctx *gor.Context) error { return c.Destroy(ctx) },
			expectedError: "Destroy action not implemented",
			setupParams: func(ctx *gor.Context) {
				ctx.Params["id"] = "999"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &BaseController{}
			ctx, w := createTestContext(tt.method, tt.path)

			// Setup params if needed
			if tt.setupParams != nil {
				tt.setupParams(ctx)
			}

			// Execute action
			err := tt.action(controller, ctx)
			if err != nil {
				t.Errorf("%s() error = %v", tt.name, err)
			}

			// Check response status
			resp := w.Result()
			if resp.StatusCode != http.StatusNotImplemented {
				t.Errorf("%s: Expected status %d, got %d", tt.name, http.StatusNotImplemented, resp.StatusCode)
			}

			// Check JSON response
			var result map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Errorf("Failed to decode response: %v", err)
			}

			if result["error"] != tt.expectedError {
				t.Errorf("Expected error message '%s', got '%s'", tt.expectedError, result["error"])
			}

			// Check ID param if it should be in response
			if strings.Contains(tt.path, "/") && tt.setupParams != nil {
				if id, ok := ctx.Params["id"]; ok {
					if result["id"] != id {
						t.Errorf("Expected id '%s' in response, got '%s'", id, result["id"])
					}
				}
			}
		})
	}
}

func TestApplicationControllerFilters(t *testing.T) {
	controller := &ApplicationController{}

	t.Run("BeforeAction", func(t *testing.T) {
		ctx, _ := createTestContext("GET", "/")
		err := controller.BeforeAction(ctx)
		if err != nil {
			t.Errorf("BeforeAction() error = %v", err)
		}
	})

	t.Run("AfterAction", func(t *testing.T) {
		ctx, _ := createTestContext("GET", "/")
		err := controller.AfterAction(ctx)
		if err != nil {
			t.Errorf("AfterAction() error = %v", err)
		}
	})
}

func TestApplicationControllerInheritance(t *testing.T) {
	controller := &ApplicationController{}
	ctx, w := createTestContext("GET", "/")

	// Test that ApplicationController inherits BaseController methods
	err := controller.Index(ctx)
	if err != nil {
		t.Errorf("Index() error = %v", err)
	}

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("Expected status %d, got %d", http.StatusNotImplemented, resp.StatusCode)
	}

	// Verify JSON response
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if result["error"] != "Index action not implemented" {
		t.Errorf("Expected error message 'Index action not implemented', got %s", result["error"])
	}
}

func TestControllerWithDifferentMethods(t *testing.T) {
	controller := &BaseController{}

	// Test with PATCH method for Update
	ctx, w := createTestContext("PATCH", "/123")
	ctx.Params["id"] = "123"

	err := controller.Update(ctx)
	if err != nil {
		t.Errorf("Update() with PATCH error = %v", err)
	}

	resp := w.Result()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("Expected status %d, got %d", http.StatusNotImplemented, resp.StatusCode)
	}
}

// Benchmark tests
func BenchmarkBaseControllerIndex(b *testing.B) {
	controller := &BaseController{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, _ := createTestContext("GET", "/")
		_ = controller.Index(ctx)
	}
}

func BenchmarkBaseControllerShow(b *testing.B) {
	controller := &BaseController{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, _ := createTestContext("GET", "/123")
		ctx.Params["id"] = "123"
		_ = controller.Show(ctx)
	}
}

func BenchmarkApplicationControllerBeforeAction(b *testing.B) {
	controller := &ApplicationController{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, _ := createTestContext("GET", "/")
		_ = controller.BeforeAction(ctx)
	}
}

func BenchmarkApplicationControllerAfterAction(b *testing.B) {
	controller := &ApplicationController{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, _ := createTestContext("GET", "/")
		_ = controller.AfterAction(ctx)
	}
}

// Example tests
func ExampleBaseController() {
	controller := &BaseController{}
	ctx, _ := createTestContext("GET", "/")

	// Call Index action
	_ = controller.Index(ctx)
	// This would return JSON with: {"error": "Index action not implemented"}
}

func ExampleApplicationController() {
	controller := &ApplicationController{}
	ctx, _ := createTestContext("GET", "/")

	// BeforeAction can be used for authentication, logging, etc.
	_ = controller.BeforeAction(ctx)

	// Execute the main action
	_ = controller.Index(ctx)

	// AfterAction can be used for cleanup, logging, etc.
	_ = controller.AfterAction(ctx)
}
