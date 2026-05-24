package execution

// ApprovalMode controls how tool execution approvals work.
type ApprovalMode string

const (
	ApprovalModeAuto    ApprovalMode = "auto"
	ApprovalModeRequire ApprovalMode = "require"
	ApprovalModeDeny    ApprovalMode = "deny"
)

// PermissionConfig controls tool/skill execution permissions per-execution.
type PermissionConfig struct {
	AllowedTools map[string]bool `json:"allowed_tools,omitempty"`
	DeniedTools  map[string]bool `json:"denied_tools,omitempty"`
	ApprovalMode ApprovalMode    `json:"approval_mode,omitempty"`
}

// IsAllowed checks if a tool/skill is permitted by this config.
func (pc *PermissionConfig) IsAllowed(name string) (bool, string) {
	if pc == nil {
		return true, ""
	}
	if len(pc.DeniedTools) > 0 && pc.DeniedTools[name] {
		return false, "denied by permission config"
	}
	if len(pc.AllowedTools) > 0 && !pc.AllowedTools[name] {
		return false, "not in allowed list"
	}
	if pc.ApprovalMode == ApprovalModeDeny {
		return false, "approval mode is deny"
	}
	return true, ""
}
