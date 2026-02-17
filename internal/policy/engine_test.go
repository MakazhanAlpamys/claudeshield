package policy

import (
	"testing"

	"github.com/MakazhanAlpamys/claudeshield/internal/config"
	"github.com/MakazhanAlpamys/claudeshield/pkg/types"
)

func TestEvaluateCommand_AllowGit(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	tests := []struct {
		cmd     string
		allowed bool
	}{
		{"git status", true},
		{"git commit -m 'test'", true},
		{"git push origin main", true},
		{"npm install", true},
		{"npm run build", true},
		{"python test.py", true},
		{"go build ./...", true},
		{"make build", true},
		{"ls -la", true},
		{"cat README.md", true},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			result := engine.EvaluateCommand(tt.cmd)
			if result.Allowed != tt.allowed {
				t.Errorf("EvaluateCommand(%q) = %v, want %v (reason: %s)", tt.cmd, result.Allowed, tt.allowed, result.Reason)
			}
		})
	}
}

func TestEvaluateCommand_BlockDangerous(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	tests := []struct {
		cmd    string
		reason string
	}{
		{"sudo apt install malware", "Privilege escalation not allowed"},
		{"rm -rf /", "Root filesystem deletion blocked"},
		{"rm -rf /*", "Root filesystem deletion blocked"},
		{"chmod 777 /etc/passwd", "Overly permissive permissions blocked"},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			result := engine.EvaluateCommand(tt.cmd)
			if result.Allowed {
				t.Errorf("EvaluateCommand(%q) should be blocked", tt.cmd)
			}
			if result.Action != types.ActionBlock {
				t.Errorf("expected ActionBlock, got %s", result.Action)
			}
		})
	}
}

func TestEvaluateCommand_BlockUnknown(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	result := engine.EvaluateCommand("some-unknown-command --flag")
	if result.Allowed {
		t.Error("unknown commands should be blocked by default (fail-secure)")
	}
}

func TestEvaluateFileAccess_BlockSensitive(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	sensitive := []string{
		"/home/user/.env",
		"/workspace/.env",
		"/home/user/.ssh/id_rsa",
		"/home/user/.aws/credentials",
	}

	for _, path := range sensitive {
		t.Run(path, func(t *testing.T) {
			result := engine.EvaluateFileAccess(path)
			if result.Allowed {
				t.Errorf("access to %q should be blocked", path)
			}
		})
	}
}

func TestEvaluateFileAccess_AllowWorkspace(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	allowed := []string{
		"/workspace/main.go",
		"/workspace/src/index.ts",
		"/workspace/README.md",
	}

	for _, path := range allowed {
		t.Run(path, func(t *testing.T) {
			result := engine.EvaluateFileAccess(path)
			if !result.Allowed {
				t.Errorf("access to %q should be allowed (reason: %s)", path, result.Reason)
			}
		})
	}
}

func TestEvaluateFileAccess_BlockOutsideWorkspace(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	result := engine.EvaluateFileAccess("/etc/passwd")
	if result.Allowed {
		t.Error("access outside workspace should be blocked")
	}
}

func TestBlockRulesTakePriority(t *testing.T) {
	cfg := &types.ProjectConfig{
		Rules: types.RulesConfig{
			Allow: []types.Rule{
				{Pattern: "rm *", Action: types.ActionAllow},
			},
			Block: []types.Rule{
				{Pattern: "rm -rf /", Action: types.ActionBlock, Reason: "blocked"},
			},
		},
	}

	engine := New(cfg)
	result := engine.EvaluateCommand("rm -rf /")

	if result.Allowed {
		t.Error("block rules should take priority over allow rules")
	}
}
