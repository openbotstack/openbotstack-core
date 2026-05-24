package providers

import "time"

// ModelEntry records a registered model's metadata for governance tracking.
type ModelEntry struct {
	ID           string    `json:"id"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	Capabilities []string  `json:"capabilities"`
	RegisteredAt time.Time `json:"registered_at"`
}

// ModelUsage records which model was used for a specific execution.
type ModelUsage struct {
	ExecutionID string    `json:"execution_id"`
	ModelID     string    `json:"model_id"`
	UsedAt      time.Time `json:"used_at"`
}

// ModelRegistry tracks registered models and their usage for governance.
type ModelRegistry interface {
	// Register records a model's metadata.
	Register(entry ModelEntry) error

	// List returns all registered models.
	List() []ModelEntry

	// Get retrieves a model by its ID.
	Get(id string) (ModelEntry, bool)

	// RecordUsage logs that a model was used for an execution.
	RecordUsage(usage ModelUsage) error

	// UsageForExecution returns the model used for a given execution.
	UsageForExecution(executionID string) (ModelUsage, bool)
}
