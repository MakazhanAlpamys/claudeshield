package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MakazhanAlpamys/claudeshield/pkg/types"
	"gopkg.in/yaml.v3"
)

const (
	ConfigFileName = ".claudeshield.yaml"
	GlobalDir      = ".claudeshield"
)

// DefaultConfig returns sensible defaults for a new project.
func DefaultConfig() *types.ProjectConfig {
	return &types.ProjectConfig{
		Sandbox: types.SandboxConfig{
			Mount:       ".:/workspace:rw",
			Network:     false,
			UseGVisor:   false,
			MemoryLimit: "2g",
			CPULimit:    2.0,
		},
		Rules: types.RulesConfig{
			Allow: []types.Rule{
				{Pattern: "git *", Action: types.ActionAllow},
				{Pattern: "npm *", Action: types.ActionAllow},
				{Pattern: "node *", Action: types.ActionAllow},
				{Pattern: "python *", Action: types.ActionAllow},
				{Pattern: "pip *", Action: types.ActionAllow},
				{Pattern: "go *", Action: types.ActionAllow},
				{Pattern: "cargo *", Action: types.ActionAllow},
				{Pattern: "make *", Action: types.ActionAllow},
				{Pattern: "cat *", Action: types.ActionAllow},
				{Pattern: "ls *", Action: types.ActionAllow},
				{Pattern: "find *", Action: types.ActionAllow},
				{Pattern: "grep *", Action: types.ActionAllow},
			},
			Block: []types.Rule{
				{Pattern: "sudo *", Action: types.ActionBlock, Reason: "Privilege escalation not allowed"},
				{Pattern: "rm -rf /", Action: types.ActionBlock, Reason: "Root filesystem deletion blocked"},
				{Pattern: "rm -rf /*", Action: types.ActionBlock, Reason: "Root filesystem deletion blocked"},
				{Pattern: "chmod 777 *", Action: types.ActionBlock, Reason: "Overly permissive permissions blocked"},
				{Pattern: "curl * | sh", Action: types.ActionBlock, Reason: "Remote code execution blocked"},
				{Pattern: "curl * | bash", Action: types.ActionBlock, Reason: "Remote code execution blocked"},
				{Pattern: "wget * | sh", Action: types.ActionBlock, Reason: "Remote code execution blocked"},
				{Pattern: "dd if=*", Action: types.ActionBlock, Reason: "Raw disk access blocked"},
				{Pattern: "mkfs.*", Action: types.ActionBlock, Reason: "Filesystem formatting blocked"},
				{Pattern: ":(){ :|:& };:", Action: types.ActionBlock, Reason: "Fork bomb blocked"},
			},
		},
		Secrets: types.SecretsConfig{
			Provider: "env",
		},
		Audit: types.AuditConfig{
			Enabled: true,
			LogDir:  ".claudeshield/logs",
		},
	}
}

// Exists checks if a config file exists in the project directory.
func Exists(projectDir string) bool {
	configPath := filepath.Join(projectDir, ConfigFileName)
	_, err := os.Stat(configPath)
	return err == nil
}

// Load reads the config from the project directory.
func Load(projectDir string) (*types.ProjectConfig, error) {
	configPath := filepath.Join(projectDir, ConfigFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", configPath, err)
	}

	return cfg, nil
}

// Save writes the config to the project directory.
func Save(projectDir string, cfg *types.ProjectConfig) error {
	configPath := filepath.Join(projectDir, ConfigFileName)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	header := []byte("# ClaudeShield configuration\n# Docs: https://github.com/MakazhanAlpamys/claudeshield\n\n")
	return os.WriteFile(configPath, append(header, data...), 0600)
}

// GlobalConfigDir returns the path to the global config directory.
func GlobalConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	dir := filepath.Join(home, GlobalDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("creating config dir: %w", err)
	}
	return dir, nil
}
