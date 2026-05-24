package skills

// ChangeEventType indicates whether a skill was registered or unregistered.
type ChangeEventType string

const (
	ChangeEventRegister   ChangeEventType = "register"
	ChangeEventUnregister ChangeEventType = "unregister"
)

// ChangeEvent describes a change to the skill registry.
type ChangeEvent struct {
	Type    ChangeEventType
	SkillID string
}

// SkillRegistry manages the catalog of available skills.
//
// The registry is a read-only view during request processing.
// Registration happens at startup or through admin operations,
// NEVER during agent execution.
type SkillRegistry interface {
	// Register adds a skill to the registry.
	// Returns error if skill with same ID already exists.
	// Thread-safe for concurrent reads after initial registration.
	Register(skill Skill) error

	// Unregister removes a skill from the registry.
	// Returns ErrSkillNotFound if the skill is not registered.
	Unregister(id string) error

	// Get retrieves a skill by ID.
	// Returns (nil, ErrSkillNotFound) if not registered.
	Get(id string) (Skill, error)

	// List returns all registered skill IDs.
	// This is used for LLM context building.
	List() []string

	// ListByPermission returns skills the caller is allowed to use.
	// Filters based on provided permission set.
	ListByPermission(permissions []string) []Skill

	// Validate checks all registered skills for consistency.
	Validate() error

	// Subscribe registers a callback invoked on register/unregister events.
	Subscribe(callback func(event ChangeEvent))
}
