package execution

import (
	"testing"
)

func TestResolveArguments_NoTemplates(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"text":  "hello world",
			"count": 42,
			"flag":  true,
		},
	}
	results := map[string]any{"prev": "value"}
	step.ResolveArguments(results)

	if step.Arguments["text"] != "hello world" {
		t.Errorf("text = %v, want hello world", step.Arguments["text"])
	}
	if step.Arguments["count"] != 42 {
		t.Errorf("count = %v, want 42", step.Arguments["count"])
	}
}

func TestResolveArguments_FullReplacement(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"val": "{{math-add}}",
		},
	}
	results := map[string]any{
		"math-add": float64(42),
	}
	step.ResolveArguments(results)

	// Full template replacement returns string since original arg was string
	val, ok := step.Arguments["val"].(string)
	if !ok {
		t.Fatalf("val type = %T, want string", step.Arguments["val"])
	}
	if val != "42" {
		t.Errorf("val = %q, want %q", val, "42")
	}
}

func TestResolveArguments_StringResult(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"text": "{{generator}}",
		},
	}
	results := map[string]any{
		"generator": "hello world",
	}
	step.ResolveArguments(results)

	if step.Arguments["text"] != "hello world" {
		t.Errorf("text = %v, want hello world", step.Arguments["text"])
	}
}

func TestResolveArguments_MapResult(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"data": "{{fetch}}",
		},
	}
	results := map[string]any{
		"fetch": map[string]any{"name": "test", "value": 99},
	}
	step.ResolveArguments(results)

	// Maps are JSON-serialized to strings when used as full template replacement
	val, ok := step.Arguments["data"].(string)
	if !ok {
		t.Fatalf("data type = %T, want string (JSON-serialized map)", step.Arguments["data"])
	}
	if val != `{"name":"test","value":99}` {
		t.Errorf("data = %q, want %q", val, `{"name":"test","value":99}`)
	}
}

func TestResolveArguments_FieldAccess(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"val": "{{math-add.result}}",
		},
	}
	results := map[string]any{
		"math-add": map[string]any{"result": float64(42)},
	}
	step.ResolveArguments(results)

	// Field access returns string since original arg was string template
	val, ok := step.Arguments["val"].(string)
	if !ok {
		t.Fatalf("val type = %T, want string", step.Arguments["val"])
	}
	if val != "42" {
		t.Errorf("val = %q, want %q", val, "42")
	}
}

func TestResolveArguments_EmbeddedTemplate(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"msg": "The answer is {{math-add}} items",
		},
	}
	results := map[string]any{
		"math-add": float64(42),
	}
	step.ResolveArguments(results)

	val, ok := step.Arguments["msg"].(string)
	if !ok {
		t.Fatalf("msg type = %T, want string", step.Arguments["msg"])
	}
	if val != "The answer is 42 items" {
		t.Errorf("msg = %q, want %q", val, "The answer is 42 items")
	}
}

func TestResolveArguments_UnresolvedReference(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"val": "{{unknown-step}}",
		},
	}
	results := map[string]any{
		"math-add": 42,
	}
	step.ResolveArguments(results)

	if step.Arguments["val"] != "{{unknown-step}}" {
		t.Errorf("val = %v, want {{unknown-step}} (unresolved)", step.Arguments["val"])
	}
}

func TestResolveArguments_MultipleTemplates(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"msg": "{{a}} plus {{b}} equals {{c}}",
		},
	}
	results := map[string]any{
		"a": float64(1),
		"b": float64(2),
		"c": float64(3),
	}
	step.ResolveArguments(results)

	val := step.Arguments["msg"]
	if val != "1 plus 2 equals 3" {
		t.Errorf("msg = %v, want %q", val, "1 plus 2 equals 3")
	}
}

func TestResolveArguments_NilArguments(t *testing.T) {
	step := &ExecutionStep{Name: "test"}
	results := map[string]any{"prev": "value"}
	step.ResolveArguments(results)
	// Should not panic
}

func TestResolveArguments_EmptyResults(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"val": "{{prev}}",
		},
	}
	step.ResolveArguments(nil)
	if step.Arguments["val"] != "{{prev}}" {
		t.Errorf("val = %v, want {{prev}} (no results to resolve)", step.Arguments["val"])
	}
}

func TestResolveArguments_NonStringArgsUntouched(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"num":   42,
			"flag":  true,
			"array": []any{1, 2, 3},
		},
	}
	results := map[string]any{"prev": "value"}
	step.ResolveArguments(results)

	if step.Arguments["num"] != 42 {
		t.Errorf("num = %v, want 42", step.Arguments["num"])
	}
	if step.Arguments["flag"] != true {
		t.Errorf("flag = %v, want true", step.Arguments["flag"])
	}
	arr, ok := step.Arguments["array"].([]any)
	if !ok || len(arr) != 3 {
		t.Errorf("array = %v, want [1 2 3]", step.Arguments["array"])
	}
}

func TestResolveArguments_FieldAccessOnNonMap(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"val": "{{math-add.field}}",
		},
	}
	results := map[string]any{
		"math-add": float64(42), // not a map, field access should fail gracefully
	}
	step.ResolveArguments(results)

	if step.Arguments["val"] != "{{math-add.field}}" {
		t.Errorf("val = %v, want {{math-add.field}} (field on non-map)", step.Arguments["val"])
	}
}

func TestCoerceStringNumbers_IntegerString(t *testing.T) {
	step := &ExecutionStep{
		Name: "math-add",
		Arguments: map[string]any{
			"a": "11050",
			"b": "500",
		},
	}
	step.CoerceStringNumbers()

	a, ok := step.Arguments["a"].(int64)
	if !ok || a != 11050 {
		t.Errorf("a = %v (%T), want int64(11050)", step.Arguments["a"], step.Arguments["a"])
	}
	b, ok := step.Arguments["b"].(int64)
	if !ok || b != 500 {
		t.Errorf("b = %v (%T), want int64(500)", step.Arguments["b"], step.Arguments["b"])
	}
}

func TestCoerceStringNumbers_FloatString(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"rate": "0.13",
		},
	}
	step.CoerceStringNumbers()

	rate, ok := step.Arguments["rate"].(float64)
	if !ok || rate != 0.13 {
		t.Errorf("rate = %v (%T), want float64(0.13)", step.Arguments["rate"], step.Arguments["rate"])
	}
}

func TestCoerceStringNumbers_BoolString(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"enabled":  "true",
			"disabled": "false",
		},
	}
	step.CoerceStringNumbers()

	if step.Arguments["enabled"] != true {
		t.Errorf("enabled = %v, want true", step.Arguments["enabled"])
	}
	if step.Arguments["disabled"] != false {
		t.Errorf("disabled = %v, want false", step.Arguments["disabled"])
	}
}

func TestCoerceStringNumbers_NonNumericUntouched(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"text":  "hello",
			"mixed": "42abc",
			"empty": "",
		},
	}
	step.CoerceStringNumbers()

	if step.Arguments["text"] != "hello" {
		t.Errorf("text = %v, want hello", step.Arguments["text"])
	}
	if step.Arguments["mixed"] != "42abc" {
		t.Errorf("mixed = %v, want 42abc", step.Arguments["mixed"])
	}
	if step.Arguments["empty"] != "" {
		t.Errorf("empty = %v, want empty string", step.Arguments["empty"])
	}
}

func TestCoerceStringNumbers_NegativeInt(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"val": "-42",
		},
	}
	step.CoerceStringNumbers()

	val, ok := step.Arguments["val"].(int64)
	if !ok || val != -42 {
		t.Errorf("val = %v (%T), want int64(-42)", step.Arguments["val"], step.Arguments["val"])
	}
}

func TestCoerceStringNumbers_AlreadyCorrectTypes(t *testing.T) {
	step := &ExecutionStep{
		Name: "test",
		Arguments: map[string]any{
			"num":   42,
			"flag":  true,
			"array": []any{1, 2},
		},
	}
	step.CoerceStringNumbers()

	if step.Arguments["num"] != 42 {
		t.Errorf("num = %v, want 42", step.Arguments["num"])
	}
	if step.Arguments["flag"] != true {
		t.Errorf("flag = %v, want true", step.Arguments["flag"])
	}
}

// Coerce+Resolve interaction: CoerceStringNumbers must run BEFORE ResolveArguments.
// If the order is reversed, template-resolved strings like "11550" would be
// incorrectly coerced to int64(11550), breaking downstream string parameters.

func TestCoerceThenResolve_StringNumberNotReCoerced(t *testing.T) {
	step := &ExecutionStep{
		Name: "step2",
		Arguments: map[string]any{
			"text": "{{step1.count}}",
		},
	}
	results := map[string]any{
		"step1": map[string]any{"count": "11550"},
	}
	// Coerce first (should do nothing — the arg is a template string, not a pure number)
	step.CoerceStringNumbers()
	// Then resolve (should produce "11550" as string from stringifyComplex)
	step.ResolveArguments(results)

	val, ok := step.Arguments["text"].(string)
	if !ok {
		t.Fatalf("text type = %T, want string", step.Arguments["text"])
	}
	if val != "11550" {
		t.Errorf("text = %q, want %q", val, "11550")
	}
}

func TestCoerceThenResolve_PureNumberArgCoercedBeforeResolve(t *testing.T) {
	step := &ExecutionStep{
		Name: "math-add",
		Arguments: map[string]any{
			"a":    "42",
			"text": "value is {{source}}",
		},
	}
	results := map[string]any{
		"source": "hello",
	}
	step.CoerceStringNumbers()
	// "42" should be coerced to int64(42)
	if step.Arguments["a"] != int64(42) {
		t.Errorf("a = %v (%T), want int64(42)", step.Arguments["a"], step.Arguments["a"])
	}
	// "value is {{source}}" should NOT be coerced (contains non-numeric chars)
	// Then resolved to "value is hello"
	step.ResolveArguments(results)
	if step.Arguments["text"] != "value is hello" {
		t.Errorf("text = %v, want 'value is hello'", step.Arguments["text"])
	}
}
