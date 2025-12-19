package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"adapter/internal/config/di"
)

// RegisterRoutes wires all HTTP routes to their handlers.
func RegisterRoutes(app *fiber.App, container *di.Container) {
	fmt.Printf("[DEBUG] RegisterRoutes called\n")
	if container == nil {
		fmt.Printf("[DEBUG] ERROR: Container is nil!\n")
		return
	}
	if container.OnSearchService == nil {
		fmt.Printf("[DEBUG] ERROR: OnSearchService is nil!\n")
		return
	}
	fmt.Printf("[DEBUG] OnSearchService is initialized, registering /on-search route\n")

	// Test endpoint to verify routing works
	app.Get("/test-on-search", func(c *fiber.Ctx) error {
		fmt.Printf("[DEBUG] Test endpoint called\n")
		return c.JSON(fiber.Map{"status": "ok", "message": "Routing works"})
	})

	onSearchHandler := NewOnSearchHandler(container.OnSearchService)
	app.Post("/on-search", onSearchHandler.HandleOnSearch)
	fmt.Printf("[DEBUG] Route /on-search registered successfully\n")
}
