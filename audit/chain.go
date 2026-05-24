package audit

// SignedEvent pairs an audit event with its chain signature.
type SignedEvent struct {
	Event     AuditEvent
	Signature string
}

// ChainVerificationResult is the outcome of verifying a signature chain.
type ChainVerificationResult struct {
	Valid        bool
	TotalChecked int
	FirstBreak   int // index of first invalid event, -1 if all valid
}
