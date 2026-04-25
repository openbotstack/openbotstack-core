package providers

// ProviderFactory creates ModelProvider instances by name.
// Centralizes provider instantiation logic so runtime code
// doesn't need to import concrete provider constructors.
type ProviderFactory struct{}

// NewProviderFactory creates a new factory.
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// Create instantiates a ModelProvider by provider name.
func (f *ProviderFactory) Create(name, baseURL, apiKey, model string) ModelProvider {
	switch name {
	case "modelscope":
		return NewModelScopeProvider(baseURL, apiKey, model)
	case "siliconflow":
		return NewSiliconFlowProvider(baseURL, apiKey, model)
	case "claude":
		return NewClaudeProvider(baseURL, apiKey, model)
	default:
		return NewOpenAIProvider(baseURL, apiKey, model)
	}
}
