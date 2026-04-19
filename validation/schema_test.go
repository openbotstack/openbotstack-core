package validation

import (
	"testing"

	csSkills "github.com/openbotstack/openbotstack-core/control/skills"
)

func TestValidateInput_NilSchema(t *testing.T) {
	err := ValidateInput([]byte(`{"key":"val"}`), nil)
	if err != nil {
		t.Errorf("nil schema should pass, got: %v", err)
	}
}

func TestValidateInput_InvalidJSON(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "object"}
	err := ValidateInput([]byte(`not json`), schema)
	if err == nil {
		t.Error("invalid JSON should fail")
	}
}

func TestValidateInput_RequiredFieldMissing(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type:     "object",
		Required: []string{"name", "age"},
	}
	err := ValidateInput([]byte(`{"name":"Alice"}`), schema)
	if err == nil {
		t.Error("missing required field 'age' should fail")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}
	if ve.Field != "age" {
		t.Errorf("expected field 'age', got %q", ve.Field)
	}
}

func TestValidateInput_WrongType(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "object"}
	err := ValidateInput([]byte(`"string"`), schema)
	if err == nil {
		t.Error("string given where object expected should fail")
	}
}

func TestValidateInput_ValidObject(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type:     "object",
		Required: []string{"text"},
		Properties: map[string]*csSkills.JSONSchema{
			"text": {Type: "string"},
		},
	}
	err := ValidateInput([]byte(`{"text":"hello"}`), schema)
	if err != nil {
		t.Errorf("valid object should pass, got: %v", err)
	}
}

func TestValidateInput_EmptyInputRequiredFields(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type:     "object",
		Required: []string{"text"},
	}
	err := ValidateInput([]byte(`{}`), schema)
	if err == nil {
		t.Error("empty object with required fields should fail")
	}
}

func TestValidateInput_NestedObjectValidation(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object",
		Properties: map[string]*csSkills.JSONSchema{
			"address": {
				Type:     "object",
				Required: []string{"city"},
				Properties: map[string]*csSkills.JSONSchema{
					"city": {Type: "string"},
				},
			},
		},
	}
	err := ValidateInput([]byte(`{"address":{}}`), schema)
	if err == nil {
		t.Error("nested missing required field should fail")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if ve.Field != "address.city" {
		t.Errorf("expected field 'address.city', got %q", ve.Field)
	}
}

func TestValidateInput_NestedObjectValid(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object",
		Properties: map[string]*csSkills.JSONSchema{
			"address": {
				Type:     "object",
				Required: []string{"city"},
				Properties: map[string]*csSkills.JSONSchema{
					"city": {Type: "string"},
				},
			},
		},
	}
	err := ValidateInput([]byte(`{"address":{"city":"NYC"}}`), schema)
	if err != nil {
		t.Errorf("valid nested object should pass, got: %v", err)
	}
}

func TestValidateInput_ArrayType(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "array"}
	err := ValidateInput([]byte(`[1,2,3]`), schema)
	if err != nil {
		t.Errorf("valid array should pass, got: %v", err)
	}

	err = ValidateInput([]byte(`"not array"`), schema)
	if err == nil {
		t.Error("string where array expected should fail")
	}
}

func TestValidateInput_BooleanType(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "boolean"}
	err := ValidateInput([]byte(`true`), schema)
	if err != nil {
		t.Errorf("valid boolean should pass, got: %v", err)
	}

	err = ValidateInput([]byte(`"yes"`), schema)
	if err == nil {
		t.Error("string where boolean expected should fail")
	}
}

func TestValidateInput_NumberType(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "number"}
	err := ValidateInput([]byte(`42.5`), schema)
	if err != nil {
		t.Errorf("valid number should pass, got: %v", err)
	}

	err = ValidateInput([]byte(`"not a number"`), schema)
	if err == nil {
		t.Error("string where number expected should fail")
	}
}

func TestValidateInput_StringType(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "string"}
	err := ValidateInput([]byte(`"hello"`), schema)
	if err != nil {
		t.Errorf("valid string should pass, got: %v", err)
	}

	err = ValidateInput([]byte(`123`), schema)
	if err == nil {
		t.Error("number where string expected should fail")
	}
}

func TestValidateInput_NilSchemaProperties(t *testing.T) {
	// Schema with type=object but no properties defined — should pass any object
	schema := &csSkills.JSONSchema{Type: "object"}
	err := ValidateInput([]byte(`{"anything":"goes"}`), schema)
	if err != nil {
		t.Errorf("object with nil properties should pass, got: %v", err)
	}
}

func TestValidateInput_PropertyWrongType(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object",
		Properties: map[string]*csSkills.JSONSchema{
			"count": {Type: "number"},
		},
	}
	err := ValidateInput([]byte(`{"count":"not a number"}`), schema)
	if err == nil {
		t.Error("wrong property type should fail")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if ve.Field != "count" {
		t.Errorf("expected field 'count', got %q", ve.Field)
	}
}

func TestValidateInput_UnsupportedSchemaType(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "integer"}
	err := ValidateInput([]byte(`42`), schema)
	if err == nil {
		t.Error("unsupported schema type should fail")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if ve.Message != `unsupported schema type "integer"` {
		t.Errorf("unexpected message: %q", ve.Message)
	}
}
