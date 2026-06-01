package skills

import (
	"testing"
)

func TestNormalizeArguments_JSONString(t *testing.T) {
	result, err := NormalizeArguments(`{"key":"value","num":42}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("expected key='value', got %v", result["key"])
	}
	if v, ok := result["num"].(float64); !ok || v != 42 {
		t.Errorf("expected num=42, got %v", result["num"])
	}
}

func TestNormalizeArguments_EmptyObject(t *testing.T) {
	result, err := NormalizeArguments(`{}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d keys", len(result))
	}
}

func TestNormalizeArguments_InvalidJSON(t *testing.T) {
	_, err := NormalizeArguments(`not json`)
	if err == nil {
		t.Error("invalid JSON should return error")
	}
}

func TestNormalizeArguments_EmptyString(t *testing.T) {
	result, err := NormalizeArguments("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map for empty string, got %d keys", len(result))
	}
}

func TestNormalizeArguments_NestedObject(t *testing.T) {
	result, err := NormalizeArguments(`{"outer":{"inner":"deep"}}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outer, ok := result["outer"].(map[string]interface{})
	if !ok {
		t.Fatal("expected nested map")
	}
	if outer["inner"] != "deep" {
		t.Errorf("expected inner=deep, got %v", outer["inner"])
	}
}

func TestNormalizeArguments_ArrayValue(t *testing.T) {
	result, err := NormalizeArguments(`{"items":[1,2,3]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := result["items"].([]interface{})
	if !ok {
		t.Fatal("expected array")
	}
	if len(arr) != 3 {
		t.Errorf("expected 3 items, got %d", len(arr))
	}
}

func TestNormalizeArguments_NullJSON(t *testing.T) {
	result, err := NormalizeArguments("null")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("null JSON should not return nil map")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d keys", len(result))
	}
}

func TestNormalizeArguments_NonObjectBoolean(t *testing.T) {
	_, err := NormalizeArguments("true")
	if err == nil {
		t.Error("boolean value should fail (not an object)")
	}
}

func TestNormalizeArguments_NonObjectNumber(t *testing.T) {
	_, err := NormalizeArguments("42")
	if err == nil {
		t.Error("number value should fail (not an object)")
	}
}

func TestNormalizeArguments_NonObjectString(t *testing.T) {
	_, err := NormalizeArguments(`"hello"`)
	if err == nil {
		t.Error("string value should fail (not an object)")
	}
}

func TestNormalizeArguments_NonObjectArray(t *testing.T) {
	_, err := NormalizeArguments(`[1,2]`)
	if err == nil {
		t.Error("array value should fail (not an object)")
	}
}
