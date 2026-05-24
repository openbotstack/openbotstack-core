package execution

import "time"

// RetryPolicy defines per-step retry behavior.
type RetryPolicy struct {
	MaxRetries     int           `json:"max_retries"`
	InitialBackoff time.Duration `json:"initial_backoff"`
	MaxBackoff     time.Duration `json:"max_backoff"`
	FailFast       bool          `json:"fail_fast"`
	FallbackTool   string        `json:"fallback_tool,omitempty"`
}

// DefaultRetryPolicy returns the standard retry configuration.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:     2,
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
		FailFast:       true,
	}
}

// StepFailure records the outcome of a failed step execution.
type StepFailure struct {
	StepID    string
	StepName  string
	Attempt   int
	Error     error
	WillRetry bool
	Fallback  bool
}
