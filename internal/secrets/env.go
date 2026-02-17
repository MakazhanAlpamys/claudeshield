package secrets

import (
	"fmt"
	"os"
	"strings"
)

// EnvProvider loads secrets from environment variables.
type EnvProvider struct{}

func (p *EnvProvider) Name() string { return "env" }

func (p *EnvProvider) Available() bool { return true }

func (p *EnvProvider) Load(keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	var missing []string

	for _, key := range keys {
		val := os.Getenv(key)
		if val == "" {
			missing = append(missing, key)
			continue
		}
		result[key] = val
	}

	if len(missing) > 0 {
		return result, fmt.Errorf("environment variables not set: %s", strings.Join(missing, ", "))
	}

	return result, nil
}
