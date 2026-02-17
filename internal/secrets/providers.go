package secrets

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// OnePasswordProvider loads secrets from 1Password CLI (op).
type OnePasswordProvider struct{}

func (p *OnePasswordProvider) Name() string { return "1password" }

func (p *OnePasswordProvider) Available() bool {
	_, err := exec.LookPath("op")
	return err == nil
}

func (p *OnePasswordProvider) Load(keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))

	for _, key := range keys {
		// key format: "vault/item/field" or just "op://vault/item/field"
		ref := key
		if !strings.HasPrefix(ref, "op://") {
			ref = "op://" + ref
		}

		out, err := exec.Command("op", "read", ref).Output()
		if err != nil {
			return result, fmt.Errorf("1password: failed to read %q: %w", key, err)
		}

		result[key] = strings.TrimSpace(string(out))
	}

	return result, nil
}

// OnePasswordEnvProvider uses "op run" to inject secrets as env vars.
type OnePasswordEnvProvider struct{}

func (p *OnePasswordEnvProvider) Name() string { return "1password-env" }

func (p *OnePasswordEnvProvider) Available() bool {
	_, err := exec.LookPath("op")
	return err == nil
}

func (p *OnePasswordEnvProvider) Load(keys []string) (map[string]string, error) {
	// Use "op inject" to resolve references in a template
	template := ""
	for _, key := range keys {
		template += fmt.Sprintf("%s={{ %s }}\n", key, key)
	}

	cmd := exec.Command("op", "inject")
	cmd.Stdin = strings.NewReader(template)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("1password inject: %w", err)
	}

	result := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}

	return result, nil
}

// VaultProvider loads secrets from HashiCorp Vault.
type VaultProvider struct{}

func (p *VaultProvider) Name() string { return "vault" }

func (p *VaultProvider) Available() bool {
	_, err := exec.LookPath("vault")
	return err == nil
}

func (p *VaultProvider) Load(keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))

	for _, key := range keys {
		// key format: "secret/data/myapp#field"
		parts := strings.SplitN(key, "#", 2)
		path := parts[0]
		field := "value"
		if len(parts) == 2 {
			field = parts[1]
		}

		out, err := exec.Command("vault", "kv", "get", "-format=json", path).Output()
		if err != nil {
			return result, fmt.Errorf("vault: failed to read %q: %w", key, err)
		}

		var resp struct {
			Data struct {
				Data map[string]interface{} `json:"data"`
			} `json:"data"`
		}
		if err := json.Unmarshal(out, &resp); err != nil {
			return result, fmt.Errorf("vault: failed to parse response for %q: %w", key, err)
		}

		val, ok := resp.Data.Data[field]
		if !ok {
			return result, fmt.Errorf("vault: field %q not found in %q", field, path)
		}

		result[key] = fmt.Sprintf("%v", val)
	}

	return result, nil
}
