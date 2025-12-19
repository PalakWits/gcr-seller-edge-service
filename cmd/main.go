package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"adapter/internal/config/di"
	"adapter/internal/handlers"
	appError "adapter/internal/shared/error"
	logger "adapter/internal/shared/log"
	"adapter/internal/shared/middleware"
)

func main() {
	ctx := context.Background()

	container, err := di.InitContainer()
	if err != nil {
		fmt.Printf("Failed to initialize container: %v\n", err)
		os.Exit(1)
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: appError.ErrorHandler(),
	})

	app.Use(middleware.RecoveryMiddleware())
	app.Use(middleware.RequestIDMiddleware())
	app.Use(middleware.LoggingMiddleware())
	app.Use(cors.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "gcr-edge-service"})
	})

	// Application routes
	handlers.RegisterRoutes(app, container)

	port := container.Config.Port

	fmt.Printf("\n🚀 Starting gcr-edge-service server\n")
	fmt.Printf("   Port: %s\n", port)
	fmt.Printf("   Logging: ✅ Enabled (Application-level logging active)\n")
	fmt.Printf("   Health Check: http://localhost:%s/health\n", port)
	fmt.Printf("\n")

	logger.Infof(ctx, "Starting gcr-edge-service server on port %s", port)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		fmt.Printf("[DEBUG] Attempting to start server on port %s\n", port)
		if err := app.Listen(":" + port); err != nil {
			fmt.Printf("[DEBUG] Server failed to start: %v\n", err)
			serverErr <- err
		}
	}()

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test if server is listening
	fmt.Printf("[DEBUG] Server should be listening on port %s\n", port)
	fmt.Printf("[DEBUG] Test with: curl http://localhost:%s/health\n", port)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		fmt.Printf("[DEBUG] Server error: %v\n", err)
		logger.Fatal(ctx, err, "Error starting server")
	case sig := <-c:
		fmt.Printf("[DEBUG] Received signal: %v\n", sig)
	}
	logger.Info(ctx, "Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := container.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, err, "Error during container shutdown")
	}

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Error(ctx, err, "Server forced to shutdown")
	} else {
		logger.Info(ctx, "Server shutdown complete")
	}
}
