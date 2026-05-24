package audit

// AuditEventMapper maps audit envelopes to industry-specific formats.
// Implementations live outside core/runtime — e.g. apps/healthcare for FHIR.
type AuditEventMapper interface {
	// Format returns the unique format identifier (e.g. "fhir_auditevent").
	Format() string

	// Map converts a single audit envelope to the target format.
	Map(envelope AuditEnvelope) (any, error)

	// MapBatch converts multiple envelopes to the target format.
	MapBatch(envelopes []AuditEnvelope) (any, error)
}
