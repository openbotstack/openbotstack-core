package capability

import (
	skills "github.com/openbotstack/openbotstack-core/control/skills"
)

// NativeToolAdapter wraps a built-in tool description as a Capability.
type NativeToolAdapter struct {
	IDVal           string
	NameVal         string
	DescVal         string
	InputSchemaVal  *skills.JSONSchema
}

func (a *NativeToolAdapter) ID() string                      { return a.IDVal }
func (a *NativeToolAdapter) Name() string                    { return a.NameVal }
func (a *NativeToolAdapter) Description() string             { return a.DescVal }
func (a *NativeToolAdapter) Kind() CapabilityKind            { return CapabilityKindNative }
func (a *NativeToolAdapter) SourceID() string                { return "builtin" }
func (a *NativeToolAdapter) InputSchema() *skills.JSONSchema { return a.InputSchemaVal }
