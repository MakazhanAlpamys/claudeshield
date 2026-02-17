package types

import "time"

// SessionState represents the current state of a sandbox session.
type SessionState string

const (
	SessionCreating  SessionState = "creating"
	SessionRunning   SessionState = "running"
	SessionPaused    SessionState = "paused"
	SessionStopped   SessionState = "stopped"
	SessionError     SessionState = "error"
)

// Session represents an active sandbox session for a Claude Code agent.
type Session struct {
	ID          string       `json:"id"`
	ProjectDir  string       `json:"project_dir"`
	ContainerID string       `json:"container_id"`
	State       SessionState `json:"state"`
	AgentName   string       `json:"agent_name"`
	WorktreeDir string       `json:"worktree_dir,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// PolicyAction defines what to do when a rule matches.
type PolicyAction string

const (
	ActionAllow PolicyAction = "allow"
	ActionBlock PolicyAction = "block"
	ActionAudit PolicyAction = "audit"
	ActionPause PolicyAction = "pause"
)

// Rule defines a single policy rule for command/file filtering.
type Rule struct {
	Pattern string       `yaml:"pattern" json:"pattern"`
	Action  PolicyAction `yaml:"action"  json:"action"`
	Reason  string       `yaml:"reason"  json:"reason,omitempty"`
}

// SandboxConfig defines sandbox isolation settings.
type SandboxConfig struct {
	Mount       string   `yaml:"mount"        json:"mount"`
	Network     bool     `yaml:"network"      json:"network"`
	ReadOnly    []string `yaml:"read_only"    json:"read_only,omitempty"`
	UseGVisor   bool     `yaml:"use_gvisor"   json:"use_gvisor"`
	MemoryLimit string   `yaml:"memory_limit" json:"memory_limit,omitempty"`
	CPULimit    float64  `yaml:"cpu_limit"    json:"cpu_limit,omitempty"`
}

// SecretsConfig defines which secrets provider to use.
type SecretsConfig struct {
	Provider string            `yaml:"provider" json:"provider"`
	Options  map[string]string `yaml:"options"  json:"options,omitempty"`
}

// AuditConfig defines audit logging settings.
type AuditConfig struct {
	Enabled  bool   `yaml:"enabled"   json:"enabled"`
	LogDir   string `yaml:"log_dir"   json:"log_dir"`
	CloudURL string `yaml:"cloud_url" json:"cloud_url,omitempty"`
}

// ProjectConfig is the top-level configuration from .claudeshield.yaml
type ProjectConfig struct {
	Sandbox SandboxConfig `yaml:"sandbox" json:"sandbox"`
	Rules   RulesConfig   `yaml:"rules"   json:"rules"`
	Secrets SecretsConfig `yaml:"secrets" json:"secrets"`
	Audit   AuditConfig   `yaml:"audit"   json:"audit"`
}

// RulesConfig groups allow and block rules.
type RulesConfig struct {
	Allow []Rule `yaml:"allow" json:"allow"`
	Block []Rule `yaml:"block" json:"block"`
}

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	Timestamp   time.Time    `json:"timestamp"`
	SessionID   string       `json:"session_id"`
	AgentName   string       `json:"agent_name"`
	EventType   string       `json:"event_type"`
	Command     string       `json:"command,omitempty"`
	FilePath    string       `json:"file_path,omitempty"`
	Action      PolicyAction `json:"action"`
	Reason      string       `json:"reason,omitempty"`
	RulePattern string       `json:"rule_pattern,omitempty"`
}

// Checkpoint represents a rollback point (Docker layer snapshot).
type Checkpoint struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	ImageID     string    `json:"image_id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// SecretProvider is the interface for secret providers.
type SecretProvider interface {
	Name() string
	Load(keys []string) (map[string]string, error)
	Available() bool
}
