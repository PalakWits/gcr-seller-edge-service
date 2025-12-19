package ports

import "context"

// SchemaValidator defines a port for validating request payloads
// against a schema (e.g. ONDC JSON Schema).
type SchemaValidator interface {
	// Validate validates the payload for a given domain and action.
	// This allows selecting different schemas for different ONDC domains.
	Validate(ctx context.Context, domain, action string, payload []byte) error
}

// ObjectStorage defines a port for uploading large payloads
// and returning the object key (path) where it was stored.
type ObjectStorage interface {
	Upload(ctx context.Context, objectName string, data []byte, contentType string) (string, error)
	GetBucket() string
}

// EventPublisher defines a port for sending events/messages
// (e.g. Kafka).
type EventPublisher interface {
	Publish(ctx context.Context, topic string, key, value []byte) error
}
