package skills

import "errors"

var (
	// ErrSkillNotFound is returned when a skill ID is not registered.
	ErrSkillNotFound = errors.New("skill: not found")

	// ErrSkillAlreadyExists is returned when registering a duplicate skill ID.
	ErrSkillAlreadyExists = errors.New("skill: already exists")

	// ErrSkillInvalid is returned when a skill fails validation.
	ErrSkillInvalid = errors.New("skill: invalid")
)
