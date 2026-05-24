package validation

import (
	"encoding/json"
	"fmt"
	"sync"
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

func TestValidateInput_IntegerType(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "integer"}

	t.Run("valid integer", func(t *testing.T) {
		err := ValidateInput([]byte(`42`), schema)
		if err != nil {
			t.Errorf("integer should be valid: %v", err)
		}
	})

	t.Run("float rejected", func(t *testing.T) {
		err := ValidateInput([]byte(`42.5`), schema)
		if err == nil {
			t.Error("float should be rejected for integer type")
		}
	})
}

func TestValidateInput_UnsupportedSchemaType(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "xml"}
	err := ValidateInput([]byte(`"data"`), schema)
	if err == nil {
		t.Error("unsupported schema type should fail")
	}
	}

// --- Extended schema tests ---

func intPtr(v int) *int            { return &v }
func floatPtr(v float64) *float64 { return &v }
func boolPtr(v bool) *bool        { return &v }

func TestValidateInput_Enum(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "string", Enum: []any{"red", "green", "blue"}}
	if err := ValidateInput([]byte(`"red"`), schema); err != nil {
		t.Errorf("valid enum: %v", err)
	}
	if err := ValidateInput([]byte(`"yellow"`), schema); err == nil {
		t.Error("invalid enum should fail")
	}
}

func TestValidateInput_MinMaxStringLength(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "string", MinLength: intPtr(2), MaxLength: intPtr(5)}
	if err := ValidateInput([]byte(`"a"`), schema); err == nil {
		t.Error("too short should fail")
	}
	if err := ValidateInput([]byte(`"abcdef"`), schema); err == nil {
		t.Error("too long should fail")
	}
	if err := ValidateInput([]byte(`"abc"`), schema); err != nil {
		t.Errorf("valid length: %v", err)
	}
	t.Run("exact minLength passes", func(t *testing.T) {
		s := &csSkills.JSONSchema{Type: "string", MinLength: intPtr(2)}
		if err := ValidateInput([]byte(`"ab"`), s); err != nil {
			t.Errorf("string at exact minLength should pass: %v", err)
		}
	})
	t.Run("exact maxLength passes", func(t *testing.T) {
		s := &csSkills.JSONSchema{Type: "string", MaxLength: intPtr(5)}
		if err := ValidateInput([]byte(`"abcde"`), s); err != nil {
			t.Errorf("string at exact maxLength should pass: %v", err)
		}
	})
}

func TestValidateInput_MinMaxNumber(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "number", Minimum: floatPtr(0), Maximum: floatPtr(100)}
	if err := ValidateInput([]byte(`-1`), schema); err == nil {
		t.Error("below minimum should fail")
	}
	if err := ValidateInput([]byte(`101`), schema); err == nil {
		t.Error("above maximum should fail")
	}
	if err := ValidateInput([]byte(`50`), schema); err != nil {
		t.Errorf("in range: %v", err)
	}
	t.Run("exact minimum passes", func(t *testing.T) {
		s := &csSkills.JSONSchema{Type: "number", Minimum: floatPtr(0)}
		if err := ValidateInput([]byte(`0`), s); err != nil {
			t.Errorf("value equal to minimum should pass: %v", err)
		}
	})
	t.Run("exact maximum passes", func(t *testing.T) {
		s := &csSkills.JSONSchema{Type: "number", Maximum: floatPtr(100)}
		if err := ValidateInput([]byte(`100`), s); err != nil {
			t.Errorf("value equal to maximum should pass: %v", err)
		}
	})
	t.Run("only minimum set", func(t *testing.T) {
		s := &csSkills.JSONSchema{Type: "number", Minimum: floatPtr(0)}
		if err := ValidateInput([]byte(`999`), s); err != nil {
			t.Errorf("value above only-minimum should pass: %v", err)
		}
	})
	t.Run("only maximum set", func(t *testing.T) {
		s := &csSkills.JSONSchema{Type: "number", Maximum: floatPtr(100)}
		if err := ValidateInput([]byte(`-999`), s); err != nil {
			t.Errorf("value below only-maximum should pass: %v", err)
		}
	})
}

func TestValidateInput_Pattern(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "string", Pattern: `^[a-z]+$`}
	if err := ValidateInput([]byte(`"hello"`), schema); err != nil {
		t.Errorf("matching: %v", err)
	}
	if err := ValidateInput([]byte(`"Hello123"`), schema); err == nil {
		t.Error("non-matching should fail")
	}
}

func TestValidateInput_ArrayItems(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "array", Items: &csSkills.JSONSchema{Type: "number"}}
	if err := ValidateInput([]byte(`[1,2,3]`), schema); err != nil {
		t.Errorf("valid items: %v", err)
	}
	if err := ValidateInput([]byte(`[1,"two",3]`), schema); err == nil {
		t.Error("invalid item should fail")
	}
}

func TestValidateInput_AdditionalPropertiesFalse(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object", AdditionalProperties: boolPtr(false),
		Properties: map[string]*csSkills.JSONSchema{"name": {Type: "string"}},
	}
	if err := ValidateInput([]byte(`{"name":"test"}`), schema); err != nil {
		t.Errorf("known only: %v", err)
	}
	if err := ValidateInput([]byte(`{"name":"test","extra":1}`), schema); err == nil {
		t.Error("unknown property should fail")
	}
}

func TestValidateInput_AnyOf(t *testing.T) {
	schema := &csSkills.JSONSchema{AnyOf: []*csSkills.JSONSchema{{Type: "string"}, {Type: "number"}}}
	if err := ValidateInput([]byte(`"hi"`), schema); err != nil {
		t.Errorf("string: %v", err)
	}
	if err := ValidateInput([]byte(`42`), schema); err != nil {
		t.Errorf("number: %v", err)
	}
	if err := ValidateInput([]byte(`[1]`), schema); err == nil {
		t.Error("array should fail anyOf(string,number)")
	}
}

func TestValidateInput_AllOf(t *testing.T) {
	schema := &csSkills.JSONSchema{AllOf: []*csSkills.JSONSchema{
		{Type: "object", Properties: map[string]*csSkills.JSONSchema{"a": {Type: "string"}}, Required: []string{"a"}},
		{Type: "object", Properties: map[string]*csSkills.JSONSchema{"b": {Type: "number"}}, Required: []string{"b"}},
	}}
	if err := ValidateInput([]byte(`{"a":"x","b":1}`), schema); err != nil {
		t.Errorf("all match: %v", err)
	}
	if err := ValidateInput([]byte(`{"a":"x"}`), schema); err == nil {
		t.Error("missing required from second should fail")
	}
}

func TestValidateInputStrict_RejectsUnknown(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object",
		Properties: map[string]*csSkills.JSONSchema{"name": {Type: "string"}},
		Required: []string{"name"},
	}
	if err := ValidateInputStrict([]byte(`{"name":"ok"}`), schema); err != nil {
		t.Errorf("exact: %v", err)
	}
	if err := ValidateInputStrict([]byte(`{"name":"ok","x":1}`), schema); err == nil {
		t.Error("strict should reject unknown")
	}
	if err := ValidateInput([]byte(`{"name":"ok","x":1}`), schema); err != nil {
		t.Errorf("non-strict allows unknown: %v", err)
	}
}

func TestValidateInput_ComplexSchema(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object", Description: "Person",
		Properties: map[string]*csSkills.JSONSchema{
			"name":  {Type: "string", MinLength: intPtr(1), MaxLength: intPtr(100)},
			"age":   {Type: "integer", Minimum: floatPtr(0), Maximum: floatPtr(150)},
			"role":  {Type: "string", Enum: []any{"admin", "member"}},
			"tags":  {Type: "array", Items: &csSkills.JSONSchema{Type: "string"}},
		},
		Required:             []string{"name", "age"},
		AdditionalProperties: boolPtr(false),
	}
	if err := ValidateInput([]byte(`{"name":"Alice","age":30,"role":"admin","tags":["dev"]}`), schema); err != nil {
		t.Errorf("complex valid: %v", err)
	}
}
func TestValidateInput_OneOf(t *testing.T) {
	schema := &csSkills.JSONSchema{OneOf: []*csSkills.JSONSchema{
		{Type: "string"},
		{Type: "number"},
	}}
	if err := ValidateInput([]byte(`"hi"`), schema); err != nil {
		t.Errorf("string should match oneOf: %v", err)
	}
	if err := ValidateInput([]byte(`42`), schema); err != nil {
		t.Errorf("number should match oneOf: %v", err)
	}
	if err := ValidateInput([]byte(`true`), schema); err == nil {
		t.Error("bool should fail oneOf(string,number)")
	}
	// Matches both -> should fail (must match exactly 1)
	dualSchema := &csSkills.JSONSchema{OneOf: []*csSkills.JSONSchema{
		{Type: "number", Minimum: floatPtr(0)},
		{Type: "number", Minimum: floatPtr(10)},
	}}
	if err := ValidateInput([]byte(`50`), dualSchema); err == nil {
		t.Error("matching both schemas in oneOf should fail")
	}
}

func TestValidateInput_EmptyEnum(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "string", Enum: []any{}}
	if err := ValidateInput([]byte(`"anything"`), schema); err != nil {
		t.Errorf("empty enum should be treated as no constraint: %v", err)
	}
}

func TestValidateInput_NilItems(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "array"}
	if err := ValidateInput([]byte(`[1,"mixed",true]`), schema); err != nil {
		t.Errorf("array with nil items should accept anything: %v", err)
	}
}

func TestValidateInput_AdditionalPropertiesTrue(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object",
		AdditionalProperties: boolPtr(true),
		Properties: map[string]*csSkills.JSONSchema{"name": {Type: "string"}},
	}
	if err := ValidateInput([]byte(`{"name":"test","extra":1}`), schema); err != nil {
		t.Errorf("additionalProperties=true should allow unknown: %v", err)
	}
}

func TestValidateInputStrict_NoProperties(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "object"}
	if err := ValidateInputStrict([]byte(`{"anything":"goes"}`), schema); err != nil {
		t.Errorf("strict with no properties should allow all keys: %v", err)
	}
}

func TestValidateInput_InvalidPattern(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "string", Pattern: "[invalid"}
	if err := ValidateInput([]byte(`"test"`), schema); err == nil {
		t.Error("invalid regex pattern should fail")
	}
}

func TestValidateInput_IntegerWithRange(t *testing.T) {
	schema := &csSkills.JSONSchema{Type: "integer", Minimum: floatPtr(1), Maximum: floatPtr(10)}
	if err := ValidateInput([]byte(`5`), schema); err != nil {
		t.Errorf("integer in range should pass: %v", err)
	}
	if err := ValidateInput([]byte(`0`), schema); err == nil {
		t.Error("integer below minimum should fail")
	}
	if err := ValidateInput([]byte(`11`), schema); err == nil {
		t.Error("integer above maximum should fail")
	}
}
func TestValidateInput_EmptyTypeWithProperties(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Properties: map[string]*csSkills.JSONSchema{
			"name": {Type: "string"},
		},
		Required: []string{"name"},
	}
	if err := ValidateInput([]byte(`{"name":"test"}`), schema); err != nil {
		t.Errorf("empty type with properties should validate: %v", err)
	}
	if err := ValidateInput([]byte(`{}`), schema); err == nil {
		t.Error("empty type with required should still enforce required")
	}
}

// --- G6: concurrent validation safety ---

func TestValidateInput_ConcurrentSafety(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object",
		Properties: map[string]*csSkills.JSONSchema{
			"name":  {Type: "string"},
			"email": {Type: "string", Pattern: "^[^@]+@[^@]+$"},
		},
		Required: []string{"name"},
	}

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			input := fmt.Sprintf(`{"name":"user%d","email":"u%d@test.com"}`, idx, idx)
			if err := ValidateInput([]byte(input), schema); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent validation failed: %v", err)
	}
}

func TestValidateInputStrict_WithComposition(t *testing.T) {
	t.Run("anyOf strict rejects unknown in sub-schema", func(t *testing.T) {
		schema := &csSkills.JSONSchema{
			AnyOf: []*csSkills.JSONSchema{
				{
					Type: "object",
					Properties: map[string]*csSkills.JSONSchema{
						"name": {Type: "string"},
					},
				},
			},
		}
		if err := ValidateInputStrict([]byte(`{"name":"ok","extra":"bad"}`), schema); err == nil {
			t.Error("strict mode should reject unknown property in anyOf sub-schema")
		}
	})

	t.Run("anyOf non-strict allows unknown in sub-schema", func(t *testing.T) {
		schema := &csSkills.JSONSchema{
			AnyOf: []*csSkills.JSONSchema{
				{
					Type: "object",
					Properties: map[string]*csSkills.JSONSchema{
						"name": {Type: "string"},
					},
				},
			},
		}
		if err := ValidateInput([]byte(`{"name":"ok","extra":"allowed"}`), schema); err != nil {
			t.Errorf("non-strict should allow unknown in anyOf sub-schema: %v", err)
		}
	})

	t.Run("allOf strict: each sub-schema independently rejects unknown", func(t *testing.T) {
		schema := &csSkills.JSONSchema{
			AllOf: []*csSkills.JSONSchema{
				{
					Type: "object",
					Properties: map[string]*csSkills.JSONSchema{
						"a": {Type: "string"},
						"b": {Type: "integer"},
					},
				},
			},
		}
		if err := ValidateInputStrict([]byte(`{"a":"x","b":1}`), schema); err != nil {
			t.Errorf("valid input should pass: %v", err)
		}
		if err := ValidateInputStrict([]byte(`{"a":"x","c":1}`), schema); err == nil {
			t.Error("strict mode should reject unknown property in allOf sub-schema")
		}
	})
}


// --- JSON Schema 2020-12: const tests ---

func TestValidateInput_Const_ValidValue(t *testing.T) {
	schema := &csSkills.JSONSchema{Const: &csSkills.ConstValue{Val: "hello"}}
	if err := ValidateInput([]byte(`"hello"`), schema); err != nil {
		t.Errorf("const match should pass: %v", err)
	}
}

func TestValidateInput_Const_InvalidValue(t *testing.T) {
	schema := &csSkills.JSONSchema{Const: &csSkills.ConstValue{Val: "hello"}}
	if err := ValidateInput([]byte(`"world"`), schema); err == nil {
		t.Error("const mismatch should fail")
	}
}

func TestValidateInput_Const_String(t *testing.T) {
	schema := &csSkills.JSONSchema{Const: &csSkills.ConstValue{Val: "exact"}}
	if err := ValidateInput([]byte(`"exact"`), schema); err != nil {
		t.Errorf("const string match: %v", err)
	}
	if err := ValidateInput([]byte(`"EXACT"`), schema); err == nil {
		t.Error("const string case mismatch should fail")
	}
}

func TestValidateInput_Const_Number(t *testing.T) {
	schema := &csSkills.JSONSchema{Const: &csSkills.ConstValue{Val: float64(42)}}
	if err := ValidateInput([]byte(`42`), schema); err != nil {
		t.Errorf("const number match: %v", err)
	}
	if err := ValidateInput([]byte(`43`), schema); err == nil {
		t.Error("const number mismatch should fail")
	}
}

func TestValidateInput_Const_Bool(t *testing.T) {
	schema := &csSkills.JSONSchema{Const: &csSkills.ConstValue{Val: true}}
	if err := ValidateInput([]byte(`true`), schema); err != nil {
		t.Errorf("const bool match: %v", err)
	}
	if err := ValidateInput([]byte(`false`), schema); err == nil {
		t.Error("const bool mismatch should fail")
	}
}

func TestValidateInput_Const_Null(t *testing.T) {
	schema := &csSkills.JSONSchema{Const: &csSkills.ConstValue{Val: nil}}
	if err := ValidateInput([]byte(`null`), schema); err != nil {
		t.Errorf("const null match: %v", err)
	}
	if err := ValidateInput([]byte(`0`), schema); err == nil {
		t.Error("const null vs 0 should fail")
	}
}

func TestValidateInput_Const_EmptyString(t *testing.T) {
	schema := &csSkills.JSONSchema{Const: &csSkills.ConstValue{Val: ""}}
	if err := ValidateInput([]byte(`""`), schema); err != nil {
		t.Errorf("const empty string match: %v", err)
	}
	if err := ValidateInput([]byte(`" "`), schema); err == nil {
		t.Error("const empty string vs space should fail")
	}
}

// --- JSON Schema 2020-12: prefixItems tests ---

func TestValidateInput_PrefixItems_ValidTuple(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "array",
		PrefixItems: []*csSkills.JSONSchema{
			{Type: "string"},
			{Type: "number"},
			{Type: "boolean"},
		},
	}
	if err := ValidateInput([]byte(`["hello",42,true]`), schema); err != nil {
		t.Errorf("valid prefixItems tuple: %v", err)
	}
}

func TestValidateInput_PrefixItems_ItemTypeMismatch(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "array",
		PrefixItems: []*csSkills.JSONSchema{
			{Type: "string"},
			{Type: "number"},
		},
	}
	if err := ValidateInput([]byte(`[42,"hello"]`), schema); err == nil {
		t.Error("prefixItems type mismatch should fail")
	}
}

func TestValidateInput_PrefixItems_FewerItemsThanPrefix(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "array",
		PrefixItems: []*csSkills.JSONSchema{
			{Type: "string"},
			{Type: "number"},
			{Type: "boolean"},
		},
	}
	// Fewer items than prefixItems is OK — prefixItems don't imply required
	if err := ValidateInput([]byte(`["hello",42]`), schema); err != nil {
		t.Errorf("fewer items than prefixItems should pass: %v", err)
	}
}

func TestValidateInput_PrefixItems_MoreItemsThanPrefix(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "array",
		PrefixItems: []*csSkills.JSONSchema{
			{Type: "string"},
			{Type: "number"},
		},
	}
	// Items beyond prefixItems are unconstrained
	if err := ValidateInput([]byte(`["hello",42,true,"extra"]`), schema); err != nil {
		t.Errorf("more items than prefixItems should pass: %v", err)
	}
}

func TestValidateInput_PrefixItems_EmptyArray(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "array",
		PrefixItems: []*csSkills.JSONSchema{
			{Type: "string"},
		},
	}
	if err := ValidateInput([]byte(`[]`), schema); err != nil {
		t.Errorf("empty array with prefixItems should pass: %v", err)
	}
}

func TestValidateInput_PrefixItems_WithItemsForRest(t *testing.T) {
	// prefixItems validates first N, Items validates the rest
	schema := &csSkills.JSONSchema{
		Type: "array",
		PrefixItems: []*csSkills.JSONSchema{
			{Type: "string"},
			{Type: "number"},
		},
		Items: &csSkills.JSONSchema{Type: "boolean"},
	}
	if err := ValidateInput([]byte(`["hello",42,true,false]`), schema); err != nil {
		t.Errorf("prefixItems + items for rest should pass: %v", err)
	}
	// Third item is not boolean — should fail via Items
	if err := ValidateInput([]byte(`["hello",42,"not-bool"]`), schema); err == nil {
		t.Error("prefixItems + items: non-bool rest item should fail")
	}
}

func TestValidateInput_PrefixItems_NilData(t *testing.T) {
	schema := &csSkills.JSONSchema{
		PrefixItems: []*csSkills.JSONSchema{
			{Type: "string"},
		},
	}
	// Not an array — prefixItems should be a no-op
	if err := ValidateInput([]byte(`"not-array"`), schema); err != nil {
		t.Errorf("prefixItems on non-array should be no-op: %v", err)
	}
}

// --- JSON Schema 2020-12: if/then/else tests ---

func TestValidateInput_IfThenElse_ConditionMet(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object",
		Properties: map[string]*csSkills.JSONSchema{
			"kind":  {Type: "string"},
			"value": {Type: "string"},
		},
		If: &csSkills.JSONSchema{
			Properties: map[string]*csSkills.JSONSchema{
				"kind": {Const: &csSkills.ConstValue{Val: "greeting"}},
			},
			Required: []string{"kind"},
		},
		Then: &csSkills.JSONSchema{
			Properties: map[string]*csSkills.JSONSchema{
				"value": {MinLength: intPtr(1)},
			},
		},
	}
	if err := ValidateInput([]byte(`{"kind":"greeting","value":"hello"}`), schema); err != nil {
		t.Errorf("if/then condition met with valid data: %v", err)
	}
}

func TestValidateInput_IfThenElse_ConditionMetThenFails(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object",
		If: &csSkills.JSONSchema{
			Properties: map[string]*csSkills.JSONSchema{
				"kind": {Const: &csSkills.ConstValue{Val: "greeting"}},
			},
			Required: []string{"kind"},
		},
		Then: &csSkills.JSONSchema{
			Properties: map[string]*csSkills.JSONSchema{
				"value": {Type: "string", MinLength: intPtr(5)},
			},
			Required: []string{"value"},
		},
	}
	// if matches (kind=greeting), but then fails (value too short)
	if err := ValidateInput([]byte(`{"kind":"greeting","value":"hi"}`), schema); err == nil {
		t.Error("if matches and then fails should produce error")
	}
}

func TestValidateInput_IfThenElse_ConditionNotMetElseFails(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object",
		If: &csSkills.JSONSchema{
			Properties: map[string]*csSkills.JSONSchema{
				"kind": {Const: &csSkills.ConstValue{Val: "greeting"}},
			},
			Required: []string{"kind"},
		},
		Else: &csSkills.JSONSchema{
			Properties: map[string]*csSkills.JSONSchema{
				"fallback": {Type: "string"},
			},
			Required: []string{"fallback"},
		},
	}
	// if does NOT match (no kind field), else requires "fallback" which is missing
	if err := ValidateInput([]byte(`{"other":"data"}`), schema); err == nil {
		t.Error("if not met and else fails should produce error")
	}
}

func TestValidateInput_IfThenElse_ConditionNotMetElsePasses(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object",
		If: &csSkills.JSONSchema{
			Properties: map[string]*csSkills.JSONSchema{
				"kind": {Const: &csSkills.ConstValue{Val: "greeting"}},
			},
			Required: []string{"kind"},
		},
		Else: &csSkills.JSONSchema{
			Properties: map[string]*csSkills.JSONSchema{
				"fallback": {Type: "string"},
			},
			Required: []string{"fallback"},
		},
	}
	// if does NOT match, else requires "fallback" which is present
	if err := ValidateInput([]byte(`{"fallback":"ok"}`), schema); err != nil {
		t.Errorf("if not met and else passes: %v", err)
	}
}

func TestValidateInput_IfThenElse_IfOnlyNoThenOrElse(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object",
		If: &csSkills.JSONSchema{
			Properties: map[string]*csSkills.JSONSchema{
				"kind": {Const: &csSkills.ConstValue{Val: "greeting"}},
			},
			Required: []string{"kind"},
		},
	}
	// if alone is a no-op — neither then nor else to apply
	if err := ValidateInput([]byte(`{"kind":"greeting"}`), schema); err != nil {
		t.Errorf("if-only should be no-op when condition met: %v", err)
	}
	if err := ValidateInput([]byte(`{"other":"data"}`), schema); err != nil {
		t.Errorf("if-only should be no-op when condition not met: %v", err)
	}
}

func TestValidateInput_IfThenElse_IfThenNoElse(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object",
		If: &csSkills.JSONSchema{
			Properties: map[string]*csSkills.JSONSchema{
				"kind": {Const: &csSkills.ConstValue{Val: "greeting"}},
			},
			Required: []string{"kind"},
		},
		Then: &csSkills.JSONSchema{
			Properties: map[string]*csSkills.JSONSchema{
				"value": {Type: "string"},
			},
			Required: []string{"value"},
		},
	}
	// if matches, then passes (value present)
	if err := ValidateInput([]byte(`{"kind":"greeting","value":"hi"}`), schema); err != nil {
		t.Errorf("if+then (no else) should pass when if matches and then passes: %v", err)
	}
	// if does NOT match — no else to apply, should pass
	if err := ValidateInput([]byte(`{"other":"data"}`), schema); err != nil {
		t.Errorf("if+then (no else) should pass when if not matched: %v", err)
	}
}

func TestValidateInput_IfThenElse_NestedConditional(t *testing.T) {
	// Outer if checks kind=greeting, then contains inner if
	schema := &csSkills.JSONSchema{
		Type: "object",
		If: &csSkills.JSONSchema{
			Properties: map[string]*csSkills.JSONSchema{
				"kind": {Const: &csSkills.ConstValue{Val: "greeting"}},
			},
			Required: []string{"kind"},
		},
		Then: &csSkills.JSONSchema{
			If: &csSkills.JSONSchema{
				Properties: map[string]*csSkills.JSONSchema{
					"formal": {Const: &csSkills.ConstValue{Val: true}},
				},
				Required: []string{"formal"},
			},
			Then: &csSkills.JSONSchema{
				Properties: map[string]*csSkills.JSONSchema{
					"value": {Type: "string", MinLength: intPtr(5)},
				},
			},
			Else: &csSkills.JSONSchema{
				Properties: map[string]*csSkills.JSONSchema{
					"value": {Type: "string", MinLength: intPtr(1)},
				},
			},
		},
	}
	// kind=greeting, formal=true, value long enough
	if err := ValidateInput([]byte(`{"kind":"greeting","formal":true,"value":"Good morning"}`), schema); err != nil {
		t.Errorf("nested conditional formal path: %v", err)
	}
	// kind=greeting, formal absent, value short ok
	if err := ValidateInput([]byte(`{"kind":"greeting","value":"hi"}`), schema); err != nil {
		t.Errorf("nested conditional informal path: %v", err)
	}
	// kind=greeting, formal=true, value too short for formal
	if err := ValidateInput([]byte(`{"kind":"greeting","formal":true,"value":"hi"}`), schema); err == nil {
		t.Error("nested conditional: formal with short value should fail")
	}
}

func TestValidateInput_IfThenElse_WithStrict(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object",
		Properties: map[string]*csSkills.JSONSchema{
			"kind":  {Type: "string"},
			"value": {Type: "string"},
		},
		If: &csSkills.JSONSchema{
			Properties: map[string]*csSkills.JSONSchema{
				"kind": {Const: &csSkills.ConstValue{Val: "greeting"}},
			},
			Required: []string{"kind"},
		},
		Then: &csSkills.JSONSchema{
			Properties: map[string]*csSkills.JSONSchema{
				"value": {Type: "string"},
			},
			Required: []string{"value"},
		},
	}
	// Strict mode: should validate then/else with strict semantics
	if err := ValidateInputStrict([]byte(`{"kind":"greeting","value":"hello"}`), schema); err != nil {
		t.Errorf("strict if/then valid: %v", err)
	}
}

// --- JSON Schema 2020-12: mixed/compatibility tests ---

func TestValidateInput_MixedDraft7And202012(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type:        "object",
		Description: "Mixed Draft 7 + 2020-12",
		Schema:      "https://json-schema.org/draft/2020-12/schema",
		Properties: map[string]*csSkills.JSONSchema{
			"role":   {Type: "string", Enum: []any{"admin", "user"}},
			"status": {Const: &csSkills.ConstValue{Val: "active"}},
			"tags": {
				Type: "array",
				PrefixItems: []*csSkills.JSONSchema{
					{Type: "string"},
				},
				Items: &csSkills.JSONSchema{Type: "string"},
			},
			"config": {
				Type: "object",
				If: &csSkills.JSONSchema{
					Properties: map[string]*csSkills.JSONSchema{
						"enabled": {Const: &csSkills.ConstValue{Val: true}},
					},
					Required: []string{"enabled"},
				},
				Then: &csSkills.JSONSchema{
					Properties: map[string]*csSkills.JSONSchema{
						"level": {Type: "string", MinLength: intPtr(1)},
					},
					Required: []string{"level"},
				},
			},
		},
		Required:             []string{"role", "status"},
		AdditionalProperties: boolPtr(false),
	}
	valid := `{"role":"admin","status":"active","tags":["dev","prod"],"config":{"enabled":true,"level":"high"}}`
	if err := ValidateInput([]byte(valid), schema); err != nil {
		t.Errorf("mixed Draft 7 + 2020-12 valid: %v", err)
	}

	// const violation: status != "active"
	if err := ValidateInput([]byte(`{"role":"admin","status":"inactive"}`), schema); err == nil {
		t.Error("const violation should fail")
	}

	// prefixItems type mismatch
	if err := ValidateInput([]byte(`{"role":"admin","status":"active","tags":[42,"prod"]}`), schema); err == nil {
		t.Error("prefixItems type mismatch should fail")
	}

	// if/then: enabled=true but missing level
	if err := ValidateInput([]byte(`{"role":"admin","status":"active","config":{"enabled":true}}`), schema); err == nil {
		t.Error("conditional if/then failure should be caught")
	}
}

func TestValidateInput_SchemaVersionDetection(t *testing.T) {
	// Schema version is purely metadata — validation should work regardless
	schema := &csSkills.JSONSchema{
		Schema: "https://json-schema.org/draft/2020-12/schema",
		Type:   "string",
		Const:  &csSkills.ConstValue{Val: "test"},
	}
	if err := ValidateInput([]byte(`"test"`), schema); err != nil {
		t.Errorf("2020-12 schema version with const: %v", err)
	}

	// Draft 7 style schema still works (no $schema field)
	draft7 := &csSkills.JSONSchema{
		Type:     "object",
		Required: []string{"name"},
		Properties: map[string]*csSkills.JSONSchema{
			"name": {Type: "string"},
		},
	}
	if err := ValidateInput([]byte(`{"name":"ok"}`), draft7); err != nil {
		t.Errorf("draft 7 style schema still works: %v", err)
	}
}

// --- G15: JSONSchema $defs serialization ---

func TestJSONSchema_DefsSerialization(t *testing.T) {
	schema := &csSkills.JSONSchema{
		Type: "object",
		Properties: map[string]*csSkills.JSONSchema{
			"ref": {Type: "string"},
		},
		Defs: map[string]*csSkills.JSONSchema{
			"address": {
				Type: "object",
				Properties: map[string]*csSkills.JSONSchema{
					"street": {Type: "string"},
					"city":   {Type: "string"},
				},
			},
		},
	}

	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded csSkills.JSONSchema
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Defs == nil {
		t.Fatal("$defs should round-trip through JSON")
	}
	if decoded.Defs["address"] == nil {
		t.Fatal("address definition should be present")
	}
	if decoded.Defs["address"].Type != "object" {
		t.Errorf("address type: expected 'object', got %q", decoded.Defs["address"].Type)
	}
}
