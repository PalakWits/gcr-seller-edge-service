package validation

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"

	"adapter/internal/ports"
)

//go:embed schemas/ret11_on_search.schema.json
var ret11OnSearchSchema []byte

//go:embed schemas/ret18_search.schema.json
var ret18SearchSchema []byte

// JSONSchemaValidator implements ports.SchemaValidator using compiled
// JSON Schemas for different ONDC domains and actions.
type JSONSchemaValidator struct {
	schemas map[string]*jsonschema.Schema
}

// schemaKey returns a lookup key for domain+action combination.
func schemaKey(domain, action string) string {
	return fmt.Sprintf("%s:%s", domain, action)
}

// NewJSONSchemaValidator compiles all embedded ONDC schemas.
func NewJSONSchemaValidator() (ports.SchemaValidator, error) {
	compiler := jsonschema.NewCompiler()
	schemas := make(map[string]*jsonschema.Schema)

	// Register RET11 on_search schema
	if err := compiler.AddResource("ret11_on_search.schema.json", strings.NewReader(string(ret11OnSearchSchema))); err != nil {
		return nil, fmt.Errorf("failed to load RET11 on_search schema: %w", err)
	}
	ret11Schema, err := compiler.Compile("ret11_on_search.schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile RET11 on_search schema: %w", err)
	}
	schemas[schemaKey("ONDC:RET11", "on_search")] = ret11Schema

	// Register RET18 search schema
	if err := compiler.AddResource("ret18_search.schema.json", strings.NewReader(string(ret18SearchSchema))); err != nil {
		return nil, fmt.Errorf("failed to load RET18 search schema: %w", err)
	}
	ret18Schema, err := compiler.Compile("ret18_search.schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile RET18 search schema: %w", err)
	}
	schemas[schemaKey("ONDC:RET18", "search")] = ret18Schema

	return &JSONSchemaValidator{schemas: schemas}, nil
}

func (v *JSONSchemaValidator) Validate(ctx context.Context, domain, action string, payload []byte) error {
	key := schemaKey(domain, action)
	schema, exists := v.schemas[key]
	if !exists {
		return fmt.Errorf("no schema found for domain=%s, action=%s", domain, action)
	}

	// Unmarshal JSON bytes into interface{} for validation
	var data interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}

	if err := schema.Validate(data); err != nil {
		return fmt.Errorf("validation failed for %s: %w", key, err)
	}
	return nil
}
