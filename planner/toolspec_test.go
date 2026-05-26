package planner

import (
	"strings"
	"testing"

	"github.com/openbotstack/openbotstack-core/capability"
	"github.com/openbotstack/openbotstack-core/control/skills"
)

// --- SchemaToToolSpec: planner uses only ID/Name/Description ---

func TestSchemaToToolSpec_BasicFields(t *testing.T) {
	desc := SkillDescriptor{
		ID:          "core/summarize",
		Name:        "Summarize",
		Description: "Summarizes text",
	}
	spec := SchemaToToolSpec(desc)
	if spec.ID != "core/summarize" {
		t.Errorf("ID = %q, want 'core/summarize'", spec.ID)
	}
	if spec.Name != "Summarize" {
		t.Errorf("Name = %q, want 'Summarize'", spec.Name)
	}
	if spec.Description != "Summarizes text" {
		t.Errorf("Description = %q, want 'Summarizes text'", spec.Description)
	}
}

func TestSchemaToToolSpec_ExtractsParameters(t *testing.T) {
	desc := SkillDescriptor{
		ID:          "core/search",
		Name:        "Search",
		Description: "Search documents",
		InputSchema: &skills.JSONSchema{
			Type: "object",
			Properties: map[string]*skills.JSONSchema{
				"query": {Type: "string"},
				"limit": {Type: "integer"},
			},
			Required: []string{"query"},
		},
	}
	spec := SchemaToToolSpec(desc)
	if len(spec.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(spec.Parameters))
	}
	if spec.Parameters["query"] != "string" {
		t.Errorf("query = %q, want 'string'", spec.Parameters["query"])
	}
	if spec.Parameters["limit"] != "integer" {
		t.Errorf("limit = %q, want 'integer'", spec.Parameters["limit"])
	}
	if len(spec.Required) != 1 || spec.Required[0] != "query" {
		t.Errorf("Required = %v, want [query]", spec.Required)
	}
}

func TestSchemaToToolSpec_DescriptionAppendsToType(t *testing.T) {
	desc := SkillDescriptor{
		ID:          "core/add",
		Name:        "Add",
		Description: "Add numbers",
		InputSchema: &skills.JSONSchema{
			Type: "object",
			Properties: map[string]*skills.JSONSchema{
				"a": {Type: "number", Description: "first operand"},
				"b": {Type: "number", Description: "second operand"},
			},
		},
	}
	spec := SchemaToToolSpec(desc)
	if spec.Parameters["a"] != "number (first operand)" {
		t.Errorf("a = %q, want 'number (first operand)'", spec.Parameters["a"])
	}
	if spec.Parameters["b"] != "number (second operand)" {
		t.Errorf("b = %q, want 'number (second operand)'", spec.Parameters["b"])
	}
}

func TestSchemaToToolSpec_NilSchema(t *testing.T) {
	desc := SkillDescriptor{
		ID:          "core/hello",
		Name:        "Hello",
		Description: "Says hello",
		InputSchema: nil,
	}
	spec := SchemaToToolSpec(desc)
	if spec.ID != "core/hello" {
		t.Errorf("ID = %q, want 'core/hello'", spec.ID)
	}
	if len(spec.Parameters) != 0 {
		t.Errorf("expected 0 parameters, got %d", len(spec.Parameters))
	}
}

func TestSchemaToToolSpec_EmptyFields(t *testing.T) {
	desc := SkillDescriptor{}
	spec := SchemaToToolSpec(desc)
	if spec.ID != "" {
		t.Errorf("expected empty ID, got %q", spec.ID)
	}
	if spec.Name != "" {
		t.Errorf("expected empty Name, got %q", spec.Name)
	}
	if spec.Description != "" {
		t.Errorf("expected empty Description, got %q", spec.Description)
	}
}

// --- FormatToolSpecs ---

func TestFormatToolSpecs_Empty(t *testing.T) {
	result := FormatToolSpecs(nil)
	if result != "" {
		t.Errorf("expected empty string for nil specs, got %q", result)
	}
}

func TestFormatToolSpecs_SingleNoParams(t *testing.T) {
	specs := []ToolSpec{
		{ID: "core/hello", Name: "Hello", Description: "Says hello"},
	}
	result := FormatToolSpecs(specs)
	if !strings.Contains(result, "- core/hello (Hello): Says hello") {
		t.Errorf("expected formatted spec, got %q", result)
	}
}

func TestFormatToolSpecs_SingleWithParams(t *testing.T) {
	specs := []ToolSpec{
		{
			ID:          "core/search",
			Name:        "Search",
			Description: "Search documents",
			Parameters:  map[string]string{"query": "string", "limit": "integer"},
			Required:    []string{"query"},
		},
	}
	result := FormatToolSpecs(specs)
	if !strings.Contains(result, "Params:") {
		t.Errorf("expected Params section, got %q", result)
	}
	if !strings.Contains(result, "limit: integer") {
		t.Errorf("expected limit parameter, got %q", result)
	}
	if !strings.Contains(result, "query: string [required]") {
		t.Errorf("expected query with [required], got %q", result)
	}
}

func TestFormatToolSpecs_SortedParams(t *testing.T) {
	specs := []ToolSpec{
		{
			ID:          "t",
			Name:        "T",
			Description: "D",
			Parameters:  map[string]string{"z_field": "string", "a_field": "integer", "m_field": "boolean"},
		},
	}
	result := FormatToolSpecs(specs)
	aIdx := strings.Index(result, "a_field")
	mIdx := strings.Index(result, "m_field")
	zIdx := strings.Index(result, "z_field")
	if aIdx >= mIdx || mIdx >= zIdx {
		t.Errorf("expected alphabetical order, got: %q", result)
	}
}

func TestFormatToolSpecs_Multiple(t *testing.T) {
	specs := []ToolSpec{
		{ID: "core/a", Name: "A", Description: "First"},
		{ID: "core/b", Name: "B", Description: "Second"},
	}
	result := FormatToolSpecs(specs)
	if !strings.Contains(result, "core/a") || !strings.Contains(result, "core/b") {
		t.Errorf("expected both specs, got %q", result)
	}
}

func TestFormatToolSpecs_NoParamsSection(t *testing.T) {
	specs := []ToolSpec{
		{ID: "core/hello", Name: "Hello", Description: "Says hello"},
	}
	result := FormatToolSpecs(specs)
	if strings.Contains(result, "Params:") {
		t.Errorf("should not have Params section when empty, got %q", result)
	}
}

// --- CapabilityToToolSpec ---

func TestCapabilityToToolSpec(t *testing.T) {
	desc := capability.CapabilityDescriptor{
		ID:          "mcp.server1.search",
		Name:        "search",
		Description: "Search for documents",
		InputSchema: &skills.JSONSchema{
			Type: "object",
			Properties: map[string]*skills.JSONSchema{
				"query": {Type: "string", Description: "search query"},
				"limit": {Type: "integer"},
			},
			Required: []string{"query"},
		},
		Kind:     string(capability.CapabilityKindMCP),
		SourceID: "server1",
	}

	spec := CapabilityToToolSpec(desc)

	if spec.ID != "mcp.server1.search" {
		t.Errorf("ID = %q", spec.ID)
	}
	if spec.Name != "search" {
		t.Errorf("Name = %q", spec.Name)
	}
	if len(spec.Parameters) != 2 {
		t.Fatalf("Parameters = %d, want 2", len(spec.Parameters))
	}
	if spec.Parameters["query"] != "string (search query)" {
		t.Errorf("query param = %q", spec.Parameters["query"])
	}
	if len(spec.Required) != 1 || spec.Required[0] != "query" {
		t.Errorf("Required = %v", spec.Required)
	}
}

func TestCapabilityToToolSpec_NoSchema(t *testing.T) {
	desc := capability.CapabilityDescriptor{
		ID:          "mcp.server1.ping",
		Name:        "ping",
		Description: "Health check",
		Kind:        string(capability.CapabilityKindMCP),
		SourceID:    "server1",
	}

	spec := CapabilityToToolSpec(desc)

	if spec.ID != "mcp.server1.ping" {
		t.Errorf("ID = %q", spec.ID)
	}
	if len(spec.Parameters) != 0 {
		t.Errorf("Parameters should be empty, got %d", len(spec.Parameters))
	}
}
