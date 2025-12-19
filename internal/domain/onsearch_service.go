package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fastjson"

	"adapter/internal/adapters/storage"
	"adapter/internal/config"
	"adapter/internal/ports"
	appError "adapter/internal/shared/error"
	logger "adapter/internal/shared/log"
)

// OnSearchService encapsulates the core application logic for handling
// ONDC on_search callbacks in a hexagonal style.
type OnSearchService struct {
	validator     ports.SchemaValidator
	storage       ports.ObjectStorage
	publisher     ports.EventPublisher
	onSearchTopic string
}

type onSearchPointer struct {
	Storage       string `json:"storage"`
	Bucket        string `json:"bucket"`
	ObjectKey     string `json:"object_key"`
	Domain        string `json:"domain"`
	Action        string `json:"action"`
	TransactionID string `json:"transaction_id"`
}

// NewOnSearchService constructs a new OnSearchService.
func NewOnSearchService(
	validator ports.SchemaValidator,
	publisher ports.EventPublisher,
	onSearchTopic string,
	cfg *config.Config,
) (*OnSearchService, error) {
	minioStorage, err := storage.NewMinIOStorage(storage.MinIOConfig{
		Endpoint:  cfg.MinIOEndpoint,
		AccessKey: cfg.MinIOAccessKey,
		SecretKey: cfg.MinIOSecretKey,
		UseSSL:    cfg.MinIOUseSSL,
		Bucket:    cfg.MinIOBucket,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO storage: %w", err)
	}
	if minioStorage == nil {
		return nil, fmt.Errorf("minio storage is nil after initialization")
	}
	return &OnSearchService{
		validator:     validator,
		storage:       minioStorage,
		publisher:     publisher,
		onSearchTopic: onSearchTopic,
	}, nil
}

// HandleOnSearch validates the payload, uploads it to object storage,
// and publishes a pointer event to Kafka.
func (s *OnSearchService) HandleOnSearch(ctx context.Context, payload []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf(ctx, fmt.Errorf("panic recovered in HandleOnSearch: %v", r), "recovered from panic")
			err = appError.NewCustomError(500, appError.ErrHTTPInternalServer.Code, "internal server error", fmt.Sprintf("%v", r))
		}
	}()

	if len(payload) == 0 {
		return appError.ErrInvalidRequestBody
	}

	logger.Info(ctx, "Step 1: Extracting context from payload using fastjson")

	// 1. Extract minimal context for routing using fastjson
	var p fastjson.Parser
	v, err := p.ParseBytes(payload)
	if err != nil {
		logger.Errorf(ctx, err, "Failed to parse JSON payload")
		return appError.NewCustomError(
			400,
			appError.ErrInvalidRequestBody.Code,
			"failed to parse ONDC payload",
			err.Error(),
		)
	}

	ctxObj := v.GetObject("context")
	if ctxObj == nil {
		logger.Warn(ctx, "Missing 'context' object in payload")
		return appError.ErrMissingRequiredField
	}

	domainVal := ctxObj.Get("domain")
	actionVal := ctxObj.Get("action")
	transactionIDVal := ctxObj.Get("transaction_id")
	messageIDVal := ctxObj.Get("message_id")

	if domainVal == nil || actionVal == nil || transactionIDVal == nil || messageIDVal == nil {
		logger.Warn(ctx, "Missing required fields in context")
		return appError.ErrMissingRequiredField
	}

	domain := string(domainVal.GetStringBytes())
	action := string(actionVal.GetStringBytes())
	transactionID := string(transactionIDVal.GetStringBytes())
	messageID := string(messageIDVal.GetStringBytes())

	if domain == "" || action == "" || transactionID == "" || messageID == "" {
		logger.Warnf(ctx, "Empty required fields: domain=%s, action=%s, transaction_id=%s, message_id=%s", domain, action, transactionID, messageID)
		return appError.ErrMissingRequiredField
	}

	logger.Infof(ctx, "Extracted context: domain=%s, action=%s, transaction_id=%s, message_id=%s", domain, action, transactionID, messageID)

	// 2. Schema validation (domain/action aware)
	logger.Infof(ctx, "Step 2: Validating payload against schema for domain=%s, action=%s", domain, action)
	if err := s.validator.Validate(ctx, domain, action, payload); err != nil {
		logger.Errorf(ctx, err, "Schema validation failed")
		return appError.NewCustomError(
			400,
			appError.ErrInvalidRequestBody.Code,
			fmt.Sprintf("schema validation failed: %v", err),
		)
	}
	logger.Info(ctx, "Schema validation passed")

	// 3. Upload raw payload to object storage
	logger.Info(ctx, "Step 3: Uploading payload to object storage")
	// Normalize domain name for path (replace colons with underscores)
	domainPath := strings.ReplaceAll(domain, ":", "_")

	// Load IST timezone
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		// Fallback to UTC if IST is not available
		logger.Warnf(ctx, "Failed to load IST timezone, falling back to UTC: %v", err)
		ist = time.UTC
	}

	objectKey := fmt.Sprintf(
		"ondc/%s/%s/%s/%s_%s.json",
		domainPath,
		action,
		time.Now().In(ist).Format("2006-01-02_15-04-05"), // User-friendly format
		transactionID,
		uuid.NewString(),
	)

	logger.Infof(ctx, "Uploading to object storage: %s", objectKey)
	uploadedObjectKey, err := s.storage.Upload(ctx, objectKey, payload, "application/json")
	if err != nil {
		logger.Errorf(ctx, err, "Failed to upload payload to object storage")
		return appError.NewCustomError(
			500,
			appError.ErrDatabaseQueryFailed.Code,
			"failed to persist on_search payload",
			err.Error(),
		)
	}
	logger.Infof(ctx, "Successfully uploaded payload, object_key: %s", uploadedObjectKey)

	// 4. Publish pointer message to Kafka
	logger.Infof(ctx, "Step 4: Publishing pointer event to Kafka topic: %s", s.onSearchTopic)
	bucket := s.storage.GetBucket()
	pointer := onSearchPointer{
		Storage:       "minio",
		Bucket:        bucket,
		ObjectKey:     uploadedObjectKey,
		Domain:        domain,
		Action:        action,
		TransactionID: transactionID,
	}

	payloadBytes, err := json.Marshal(pointer)
	if err != nil {
		logger.Errorf(ctx, err, "Failed to serialize pointer")
		return appError.NewCustomError(
			500,
			appError.ErrHTTPInternalServer.Code,
			"failed to serialize on_search pointer",
			err.Error(),
		)
	}

	if err := s.publisher.Publish(ctx, s.onSearchTopic, []byte(transactionID), payloadBytes); err != nil {
		logger.Errorf(ctx, err, "Failed to publish pointer to Kafka")
		return appError.NewCustomError(
			500,
			appError.ErrHTTPInternalServer.Code,
			"failed to publish on_search pointer",
			err.Error(),
		)
	}
	logger.Info(ctx, "Successfully published pointer event to Kafka")

	return nil
}
