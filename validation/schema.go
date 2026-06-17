package validation

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/openbotstack/openbotstack-core/ai/types"
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
func ValidateInput(input []byte, schema *types.JSONSchema) error {
	if schema == nil {
		return nil
	}

	var data interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		return fmt.Errorf("%w: invalid JSON: %v", ErrSchemaValidation, err)
	}

	return validateValue(data, schema, "")
}

// ValidateValue validates an already-decoded Go value against a JSONSchema,
// without a JSON byte round-trip. Returns nil if schema is nil.
//
// This is the Verify engine for StepResult.Output (ADR-036 Phase 1): Output is a
// Go value (often map[string]any or a string), not raw JSON, so validating it
// directly avoids re-marshalling and preserves native numeric types. It reuses
// the same deterministic validator as ValidateInput — zero LLM, pure code.
func ValidateValue(data any, schema *types.JSONSchema) error {
	if schema == nil {
		return nil
	}
	return validateValue(data, schema, "")
}

// ValidateInputStrict validates with strict mode: rejects unknown properties,
// enforces all types exactly, and requires all schema constraints.
func ValidateInputStrict(input []byte, schema *types.JSONSchema) error {
	if schema == nil {
		return nil
	}

	var data interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		return fmt.Errorf("%w: invalid JSON: %v", ErrSchemaValidation, err)
	}

	return validateValueStrict(data, schema, "")
}

func validateValue(data interface{}, schema *types.JSONSchema, path string) error {
	if schema == nil {
		return nil
	}

	if schema.Type != "" {
		if err := validateType(data, schema.Type, path); err != nil {
			return err
		}
	}

	// Enum constraint
	if len(schema.Enum) > 0 {
		if err := validateEnum(data, schema.Enum, path); err != nil {
			return err
		}
	}

	// Const constraint (JSON Schema 2020-12)
	if schema.Const != nil {
		if err := validateConst(data, schema.Const.Val, path); err != nil {
			return err
		}
	}

	// Type-specific constraints
	switch schema.Type {
	case "string":
		if err := validateString(data, schema, path); err != nil {
			return err
		}
	case "number", "integer":
		if err := validateNumber(data, schema, path); err != nil {
			return err
		}
	case "array":
		if err := validateArray(data, schema, path); err != nil {
			return err
		}
	case "object", "":
		if schema.Type == "object" || (schema.Type == "" && schema.Properties != nil) {
			if err := validateObject(data, schema, path, false); err != nil {
				return err
			}
		}
	}

	// Composition (anyOf, oneOf, allOf)
	if err := validateComposition(data, schema, path, false); err != nil {
		return err
	}

	// Conditional if/then/else (JSON Schema 2020-12)
	if schema.If != nil {
		if err := validateConditional(data, schema, path, false); err != nil {
			return err
		}
	}

	return nil
}

func validateValueStrict(data interface{}, schema *types.JSONSchema, path string) error {
	if schema == nil {
		return nil
	}

	if schema.Type != "" {
		if err := validateType(data, schema.Type, path); err != nil {
			return err
		}
	}

	if len(schema.Enum) > 0 {
		if err := validateEnum(data, schema.Enum, path); err != nil {
			return err
		}
	}

	// Const constraint (JSON Schema 2020-12)
	if schema.Const != nil {
		if err := validateConst(data, schema.Const.Val, path); err != nil {
			return err
		}
	}

	switch schema.Type {
	case "string":
		if err := validateString(data, schema, path); err != nil {
			return err
		}
	case "number", "integer":
		if err := validateNumber(data, schema, path); err != nil {
			return err
		}
	case "array":
		if err := validateArrayStrict(data, schema, path); err != nil {
			return err
		}
	case "object", "":
		if schema.Type == "object" || (schema.Type == "" && schema.Properties != nil) {
			// Strict: reject unknown properties
			if err := validateObject(data, schema, path, true); err != nil {
				return err
			}
		}
	}

	if err := validateComposition(data, schema, path, true); err != nil {
		return err
	}

	// Conditional if/then/else (JSON Schema 2020-12)
	if schema.If != nil {
		if err := validateConditional(data, schema, path, true); err != nil {
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
	case "integer":
		f, ok := data.(float64)
		if !ok || f != float64(int64(f)) {
			return &ValidationError{Field: path, Message: fmt.Sprintf("expected integer, got %T", data)}
		}
	default:
		return &ValidationError{Field: path, Message: fmt.Sprintf("unsupported schema type %q", expected)}
	}
	return nil
}

func validateEnum(data interface{}, enum []any, path string) error {
	for _, v := range enum {
		if jsonEqual(data, v) {
			return nil
		}
	}
	return &ValidationError{Field: path, Message: fmt.Sprintf("value not in enum: %v", enum)}
}

func jsonEqual(a, b interface{}) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

func validateString(data interface{}, schema *types.JSONSchema, path string) error {
	s, ok := data.(string)
	if !ok {
		return nil // type mismatch already caught
	}

	if schema.MinLength != nil && len(s) < *schema.MinLength {
		return &ValidationError{Field: path, Message: fmt.Sprintf("string length %d < minLength %d", len(s), *schema.MinLength)}
	}
	if schema.MaxLength != nil && len(s) > *schema.MaxLength {
		return &ValidationError{Field: path, Message: fmt.Sprintf("string length %d > maxLength %d", len(s), *schema.MaxLength)}
	}
	if schema.Pattern != "" {
		re, err := regexp.Compile(schema.Pattern)
		if err != nil {
			return &ValidationError{Field: path, Message: fmt.Sprintf("invalid pattern %q: %v", schema.Pattern, err)}
		}
		if !re.MatchString(s) {
			return &ValidationError{Field: path, Message: fmt.Sprintf("string does not match pattern %q", schema.Pattern)}
		}
	}
	return nil
}

func validateNumber(data interface{}, schema *types.JSONSchema, path string) error {
	f, ok := data.(float64)
	if !ok {
		return nil // type mismatch already caught
	}

	if schema.Minimum != nil && f < *schema.Minimum {
		return &ValidationError{Field: path, Message: fmt.Sprintf("value %v < minimum %v", f, *schema.Minimum)}
	}
	if schema.Maximum != nil && f > *schema.Maximum {
		return &ValidationError{Field: path, Message: fmt.Sprintf("value %v > maximum %v", f, *schema.Maximum)}
	}
	return nil
}

func validateArray(data interface{}, schema *types.JSONSchema, path string) error {
	arr, ok := data.([]interface{})
	if !ok {
		return nil // type mismatch already caught
	}

	// prefixItems: tuple-style validation for first N items (JSON Schema 2020-12)
	if len(schema.PrefixItems) > 0 {
		if err := validatePrefixItems(arr, schema, path, false); err != nil {
			return err
		}
	}

	// Items: validates remaining items after prefixItems, or all items if no prefixItems
	if schema.Items != nil {
		startIdx := 0
		if len(schema.PrefixItems) > 0 {
			startIdx = len(schema.PrefixItems)
		}
		for i := startIdx; i < len(arr); i++ {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			if err := validateValue(arr[i], schema.Items, itemPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateArrayStrict(data interface{}, schema *types.JSONSchema, path string) error {
	arr, ok := data.([]interface{})
	if !ok {
		return nil
	}

	// prefixItems: tuple-style validation for first N items (JSON Schema 2020-12)
	if len(schema.PrefixItems) > 0 {
		if err := validatePrefixItems(arr, schema, path, true); err != nil {
			return err
		}
	}

	// Items: validates remaining items after prefixItems, or all items if no prefixItems
	if schema.Items != nil {
		startIdx := 0
		if len(schema.PrefixItems) > 0 {
			startIdx = len(schema.PrefixItems)
		}
		for i := startIdx; i < len(arr); i++ {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			if err := validateValueStrict(arr[i], schema.Items, itemPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateObject(data interface{}, schema *types.JSONSchema, path string, strict bool) error {
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
		if strict {
			if err := validateValueStrict(val, propSchema, fieldPath); err != nil {
				return err
			}
		} else {
			if err := validateValue(val, propSchema, fieldPath); err != nil {
				return err
			}
		}
	}

	// Strict mode or AdditionalProperties=false: reject unknown keys
	rejectUnknown := strict
	if schema.AdditionalProperties != nil && !*schema.AdditionalProperties {
		rejectUnknown = true
	}
	if rejectUnknown && len(schema.Properties) > 0 {
		for key := range obj {
			if _, known := schema.Properties[key]; !known {
				fieldPath := dotPath(path, key)
				return &ValidationError{Field: fieldPath, Message: "unknown property"}
			}
		}
	}

	return nil
}

func validateComposition(data interface{}, schema *types.JSONSchema, path string, strict bool) error {
	validate := validateValue
	if strict {
		validate = validateValueStrict
	}

	if len(schema.AnyOf) > 0 {
		matched := false
		for _, sub := range schema.AnyOf {
			if validate(data, sub, path) == nil {
				matched = true
				break
			}
		}
		if !matched {
			return &ValidationError{Field: path, Message: "value does not match any of anyOf schemas"}
		}
	}

	if len(schema.OneOf) > 0 {
		count := 0
		for _, sub := range schema.OneOf {
			if validate(data, sub, path) == nil {
				count++
			}
		}
		if count != 1 {
			return &ValidationError{Field: path, Message: fmt.Sprintf("value matches %d of oneOf schemas, expected exactly 1", count)}
		}
	}

	if len(schema.AllOf) > 0 {
		for _, sub := range schema.AllOf {
			if err := validate(data, sub, path); err != nil {
				return &ValidationError{Field: path, Message: fmt.Sprintf("value does not match allOf: %v", err)}
			}
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

// validateConst checks that data matches a single constant value (JSON Schema 2020-12).
func validateConst(data interface{}, constVal interface{}, path string) error {
	if !jsonEqual(data, constVal) {
		return &ValidationError{Field: path, Message: fmt.Sprintf("value must be %v", constVal)}
	}
	return nil
}

// validatePrefixItems validates tuple-style array items (JSON Schema 2020-12).
// Each prefixItem schema applies to the array element at the corresponding index.
// Fewer array elements than prefixItems is acceptable (prefixItems don't imply required).
func validatePrefixItems(arr []interface{}, schema *types.JSONSchema, path string, strict bool) error {
	validate := validateValue
	if strict {
		validate = validateValueStrict
	}
	for i, itemSchema := range schema.PrefixItems {
		if i >= len(arr) {
			break
		}
		itemPath := fmt.Sprintf("%s[%d]", path, i)
		if err := validate(arr[i], itemSchema, itemPath); err != nil {
			return err
		}
	}
	return nil
}

// validateConditional applies if/then/else conditional validation (JSON Schema 2020-12).
// If the "if" schema matches the data, "then" is applied; otherwise "else" is applied.
func validateConditional(data interface{}, schema *types.JSONSchema, path string, strict bool) error {
	validate := validateValue
	if strict {
		validate = validateValueStrict
	}

	ifErr := validate(data, schema.If, path)
	ifMatches := ifErr == nil
	if ifMatches && schema.Then != nil {
		thenErr := validate(data, schema.Then, path)
		if thenErr != nil {
			return &ValidationError{Field: path, Message: fmt.Sprintf("condition met but 'then' failed: %v", thenErr)}
		}
	}
	if !ifMatches && schema.Else != nil {
		if err := validate(data, schema.Else, path); err != nil {
			return &ValidationError{Field: path, Message: fmt.Sprintf("condition not met but 'else' failed: %v", err)}
		}
	}
	return nil
}
