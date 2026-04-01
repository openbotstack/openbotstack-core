package assistant

// AssistantProfile defines the static metadata for an assistant.
// It represents the "identity" of the assistant, which can be reused
// across multiple sessions and tenants if permitted.
type AssistantProfile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}
