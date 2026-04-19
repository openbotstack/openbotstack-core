// Package validation provides JSON Schema validation for skill input parameters.
package validation

import (
	"encoding/json"
	"fmt"

	csSkills "github.com/openbotstack/openbotstack-core/control/skills"
)

// ErrSchemaValidation indicates input failed JSON Schema validation.
var ErrSchemaValidation = fmt.Errorf("validation: schema validation failed")

// ValidationError contains detailed validation failure information.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("field %q: %s", e.Field, e.Message)
}

// ValidateInput validates raw JSON input against a JSONSchema.
// Returns nil if schema is nil (no schema = no validation required).
func ValidateInput(input []byte, schema *csSkills.JSONSchema) error {
	if schema == nil {
		return nil
	}

	var data interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		return fmt.Errorf("%w: invalid JSON: %v", ErrSchemaValidation, err)
	}

	return validateValue(data, schema, "")
}

func validateValue(data interface{}, schema *csSkills.JSONSchema, path string) error {
	if schema == nil {
		return nil
	}

	if schema.Type != "" {
		if err := validateType(data, schema.Type, path); err != nil {
			return err
		}
	}

	// Validate object properties and required fields
	if schema.Type == "object" || (schema.Type == "" && schema.Properties != nil) {
		if err := validateObject(data, schema, path); err != nil {
			return err
		}
	}

	return nil
}

func validateType(data interface{}, expected string, path string) error {
	switch expected {
	case "object":
		if _, ok := data.(map[string]interface{}); !ok {
			return &ValidationError{Field: path, Message: fmt.Sprintf("expected object, got %T", data)}
		}
	case "array":
		if _, ok := data.([]interface{}); !ok {
			return &ValidationError{Field: path, Message: fmt.Sprintf("expected array, got %T", data)}
		}
	case "string":
		if _, ok := data.(string); !ok {
			return &ValidationError{Field: path, Message: fmt.Sprintf("expected string, got %T", data)}
		}
	case "number":
		if _, ok := data.(float64); !ok {
			return &ValidationError{Field: path, Message: fmt.Sprintf("expected number, got %T", data)}
		}
	case "boolean":
		if _, ok := data.(bool); !ok {
			return &ValidationError{Field: path, Message: fmt.Sprintf("expected boolean, got %T", data)}
		}
	default:
		return &ValidationError{Field: path, Message: fmt.Sprintf("unsupported schema type %q", expected)}
	}
	return nil
}

func validateObject(data interface{}, schema *csSkills.JSONSchema, path string) error {
	obj, ok := data.(map[string]interface{})
	if !ok {
		return nil // type mismatch already caught by validateType
	}

	// Check required fields
	for _, req := range schema.Required {
		if _, exists := obj[req]; !exists {
			fieldPath := dotPath(path, req)
			return &ValidationError{Field: fieldPath, Message: "required field missing"}
		}
	}

	// Validate properties recursively
	for key, propSchema := range schema.Properties {
		val, exists := obj[key]
		if !exists {
			continue // missing optional property is ok
		}
		fieldPath := dotPath(path, key)
		if err := validateValue(val, propSchema, fieldPath); err != nil {
			return err
		}
	}

	return nil
}

func dotPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}
