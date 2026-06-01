package execution

import (
	"errors"
)

var (
	// ErrSkillNotLoaded is returned when executing an unloaded skills.
	ErrSkillNotLoaded = errors.New("runtime: skill not loaded")

	// ErrExecutionTimeout is returned when execution exceeds timeout.
	ErrExecutionTimeout = errors.New("runtime: execution timeout")

	// ErrResourceExhausted is returned when resource limits are exceeded.
	ErrResourceExhausted = errors.New("runtime: resource exhausted")

	// ErrPolicyRejected is returned when policy denies execution.
	ErrPolicyRejected = errors.New("runtime: policy rejected execution")
)
