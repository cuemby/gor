package controllers

import (
	"net/http"

	"github.com/cuemby/gor/pkg/gor"
)

// BaseController provides default implementations for controller actions
type BaseController struct{}

// Index lists all resources (GET /)
func (c *BaseController) Index(ctx *gor.Context) error {
	return ctx.JSON(http.StatusNotImplemented, map[string]string{
		"error": "Index action not implemented",
	})
}

// Show displays a specific resource (GET /:id)
func (c *BaseController) Show(ctx *gor.Context) error {
	return ctx.JSON(http.StatusNotImplemented, map[string]string{
		"error": "Show action not implemented",
		"id":    ctx.Param("id"),
	})
}

// New displays form for creating new resource (GET /new)
func (c *BaseController) New(ctx *gor.Context) error {
	return ctx.JSON(http.StatusNotImplemented, map[string]string{
		"error": "New action not implemented",
	})
}

// Create processes creation of new resource (POST /)
func (c *BaseController) Create(ctx *gor.Context) error {
	return ctx.JSON(http.StatusNotImplemented, map[string]string{
		"error": "Create action not implemented",
	})
}

// Edit displays form for editing resource (GET /:id/edit)
func (c *BaseController) Edit(ctx *gor.Context) error {
	return ctx.JSON(http.StatusNotImplemented, map[string]string{
		"error": "Edit action not implemented",
		"id":    ctx.Param("id"),
	})
}

// Update processes resource updates (PUT/PATCH /:id)
func (c *BaseController) Update(ctx *gor.Context) error {
	return ctx.JSON(http.StatusNotImplemented, map[string]string{
		"error": "Update action not implemented",
		"id":    ctx.Param("id"),
	})
}

// Destroy deletes a resource (DELETE /:id)
func (c *BaseController) Destroy(ctx *gor.Context) error {
	return ctx.JSON(http.StatusNotImplemented, map[string]string{
		"error": "Destroy action not implemented",
		"id":    ctx.Param("id"),
	})
}

// ApplicationController is the base controller for the application
// All application controllers should embed this
type ApplicationController struct {
	BaseController
}

// BeforeAction runs before any controller action
func (c *ApplicationController) BeforeAction(ctx *gor.Context) error {
	// Add any application-wide before filters here
	// e.g., authentication, authorization, logging
	return nil
}

// AfterAction runs after any controller action
func (c *ApplicationController) AfterAction(ctx *gor.Context) error {
	// Add any application-wide after filters here
	// e.g., cleanup, logging
	return nil
}
