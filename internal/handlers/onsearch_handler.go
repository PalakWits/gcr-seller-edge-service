package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"adapter/internal/domain"
	appError "adapter/internal/shared/error"
	logger "adapter/internal/shared/log"
)

type OnSearchHandler struct {
	service *domain.OnSearchService
}

func NewOnSearchHandler(service *domain.OnSearchService) *OnSearchHandler {
	fmt.Printf("[DEBUG] NewOnSearchHandler created with service: %p\n", service)
	return &OnSearchHandler{service: service}
}

// HandleOnSearch is the HTTP adapter for the /on-search endpoint.
// It accepts a heavy ONDC payload, performs schema validation and then
// delegates to the domain service to persist and publish a pointer.
func (h *OnSearchHandler) HandleOnSearch(c *fiber.Ctx) error {
	// Immediate console output to verify handler is called
	fmt.Printf("\n[DEBUG] HandleOnSearch called - Method: %s, Path: %s\n", c.Method(), c.Path())
	fmt.Printf("[DEBUG] Handler's service object: %p\n", h.service)

	ctx := c.UserContext()
	if ctx == nil {
		fmt.Printf("[DEBUG] UserContext is nil, creating background context\n")
		ctx = context.Background()
	}

	// Add timeout to prevent hanging requests
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	bodySize := len(c.Body())
	fmt.Printf("[DEBUG] Request body size: %d bytes\n", bodySize)
	logger.Infof(ctx, "Received /on-search request, body size: %d bytes", bodySize)

	if h.service == nil {
		fmt.Printf("[DEBUG] ERROR: service is nil\n")
		logger.Error(ctx, fmt.Errorf("service is nil"), "OnSearchService not initialized")
		return appError.ErrHTTPInternalServer
	}

	fmt.Printf("[DEBUG] Reading request body...\n")
	payload := c.Body()
	fmt.Printf("[DEBUG] Body read complete, size: %d\n", len(payload))

	if len(payload) == 0 {
		fmt.Printf("[DEBUG] Empty request body\n")
		logger.Warn(ctx, "Empty request body received")
		return appError.ErrInvalidRequestBody
	}

	logger.Infof(ctx, "Processing on-search payload, size: %d bytes", len(payload))

	err := h.service.HandleOnSearch(ctx, payload)
	if err != nil {
		logger.Errorf(ctx, err, "Failed to handle on-search request")
		return err
	}

	logger.Info(ctx, "Successfully processed on-search request")

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"status": "accepted",
	})
}
