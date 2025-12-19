package di

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"adapter/internal/adapters/messaging"
	"adapter/internal/adapters/validation"
	"adapter/internal/config"
	"adapter/internal/domain"
	db "adapter/internal/shared/database"
	logger "adapter/internal/shared/log"
)

type Container struct {
	Config          *config.Config
	DB              *gorm.DB
	OnSearchService *domain.OnSearchService
}

func (c *Container) Shutdown(ctx context.Context) error {
	logger.Info(ctx, "Shutting down container resources...")

	if c.DB != nil {
		if err := db.Close(); err != nil {
			logger.Error(ctx, err, "Failed to close database connection")
		}
	}

	logger.Info(ctx, "Container shutdown complete")
	return nil
}

func InitContainer() (*Container, error) {
	fmt.Printf("[DEBUG] Starting container initialization...\n")
	cfg, err := config.LoadConfig()
	ctx := context.Background()
	if err != nil {
		fmt.Printf("[DEBUG] Config load failed: %v\n", err)
		logger.Fatal(ctx, fmt.Errorf("failed to load config: %w", err), "Configuration error")
	}
	fmt.Printf("[DEBUG] Config loaded successfully\n")

	fmt.Printf("[DEBUG] Initializing database...\n")
	database, err := db.Init(cfg.DatabaseURL)
	if err != nil {
		fmt.Printf("[DEBUG] Database init failed: %v\n", err)
		logger.Fatal(ctx, fmt.Errorf("failed to initialize database: %w", err), "Database initialization error")
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	fmt.Printf("[DEBUG] Database initialized successfully\n")

	// Run database migrations using golang-migrate only
	logger.Info(ctx, "Running database migrations...")
	if err := MigrateDB(database); err != nil {
		logger.Fatal(ctx, err, "Failed to run database migrations")
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}
	logger.Info(ctx, "Database migrations completed successfully")

	// Kafka publisher adapter
	fmt.Printf("[DEBUG] Initializing Kafka publisher...\n")
	kafkaPublisher, err := messaging.NewKafkaPublisher(messaging.KafkaConfig{
		Brokers: cfg.KafkaBrokers,
	})
	if err != nil {
		fmt.Printf("[DEBUG] Kafka init failed: %v\n", err)
		logger.Fatal(ctx, fmt.Errorf("failed to initialize Kafka publisher: %w", err), "Kafka initialization error")
	}
	fmt.Printf("[DEBUG] Kafka publisher initialized successfully\n")

	// JSON schema validator for ONDC RET11 on_search
	fmt.Printf("[DEBUG] Initializing schema validator...\n")
	schemaValidator, err := validation.NewJSONSchemaValidator()
	if err != nil {
		fmt.Printf("[DEBUG] Schema validator init failed: %v\n", err)
		logger.Fatal(ctx, fmt.Errorf("failed to initialize schema validator: %w", err), "Schema validation initialization error")
	}
	fmt.Printf("[DEBUG] Schema validator initialized successfully\n")

	onSearchService, err := domain.NewOnSearchService(
		schemaValidator,
		kafkaPublisher,
		cfg.KafkaOnSearchTopic,
		cfg,
	)
	if err != nil {
		logger.Fatal(ctx, fmt.Errorf("failed to create OnSearchService: %w", err), "OnSearchService initialization error")
	}

	return &Container{
		Config:          cfg,
		DB:              database,
		OnSearchService: onSearchService,
	}, err
}
