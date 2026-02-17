package policy

import (
	"path/filepath"
	"strings"

	"github.com/MakazhanAlpamys/claudeshield/pkg/types"
)

// Engine evaluates commands and file accesses against the policy rules.
type Engine struct {
	config *types.ProjectConfig
}

// New creates a new policy engine with the given config.
func New(cfg *types.ProjectConfig) *Engine {
	return &Engine{config: cfg}
}

// Result contains the outcome of a policy evaluation.
type Result struct {
	Allowed bool
	Action  types.PolicyAction
	Rule    *types.Rule
	Reason  string
}

// EvaluateCommand checks if a command is allowed by the policy.
func (e *Engine) EvaluateCommand(command string) Result {
	command = strings.TrimSpace(command)

	// Check block rules first (deny takes priority)
	for i, rule := range e.config.Rules.Block {
		if matchPattern(rule.Pattern, command) {
			return Result{
				Allowed: false,
				Action:  types.ActionBlock,
				Rule:    &e.config.Rules.Block[i],
				Reason:  rule.Reason,
			}
		}
	}

	// Check allow rules
	for i, rule := range e.config.Rules.Allow {
		if matchPattern(rule.Pattern, command) {
			return Result{
				Allowed: true,
				Action:  types.ActionAllow,
				Rule:    &e.config.Rules.Allow[i],
			}
		}
	}

	// Default: block unknown commands (fail-secure)
	return Result{
		Allowed: false,
		Action:  types.ActionBlock,
		Reason:  "Command not in allowlist",
	}
}

// EvaluateFileAccess checks if access to a file path is allowed.
func (e *Engine) EvaluateFileAccess(path string) Result {
	// Block sensitive files
	sensitivePatterns := []string{
		"*/.env",
		"*/.env.*",
		"*/.ssh/*",
		"*/.aws/*",
		"*/.gnupg/*",
		"*/.docker/config.json",
		"*/id_rsa",
		"*/id_ed25519",
		"*/.bash_history",
		"*/.zsh_history",
		"*/.gitconfig",
		"*/.npmrc",
		"*/.pypirc",
	}

	for _, pattern := range sensitivePatterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return Result{
				Allowed: false,
				Action:  types.ActionBlock,
				Reason:  "Access to sensitive file blocked: " + path,
			}
		}
	}

	// Allow access to workspace
	if strings.HasPrefix(path, "/workspace") {
		return Result{
			Allowed: true,
			Action:  types.ActionAllow,
		}
	}

	// Block everything outside workspace
	return Result{
		Allowed: false,
		Action:  types.ActionBlock,
		Reason:  "Access outside workspace blocked: " + path,
	}
}

// matchPattern performs glob-style pattern matching on commands.
func matchPattern(pattern, command string) bool {
	// Handle simple wildcard at the end: "git *" matches "git status"
	if strings.HasSuffix(pattern, " *") {
		prefix := strings.TrimSuffix(pattern, " *")
		if command == prefix || strings.HasPrefix(command, prefix+" ") {
			return true
		}
	}

	// Handle wildcard prefix: "*.py" matches anything ending in .py
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(command, suffix)
	}

	// Exact match
	if command == pattern {
		return true
	}

	// Glob match
	matched, _ := filepath.Match(pattern, command)
	return matched
}
