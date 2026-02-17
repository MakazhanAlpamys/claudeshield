package secrets

import (
	"fmt"

	"github.com/MakazhanAlpamys/claudeshield/pkg/types"
)

// Registry manages available secret providers.
type Registry struct {
	providers map[string]types.SecretProvider
}

// NewRegistry creates a registry with all built-in providers.
func NewRegistry() *Registry {
	r := &Registry{
		providers: make(map[string]types.SecretProvider),
	}

	r.Register(&EnvProvider{})
	r.Register(&OnePasswordProvider{})
	r.Register(&OnePasswordEnvProvider{})
	r.Register(&VaultProvider{})

	return r
}

// Register adds a provider to the registry.
func (r *Registry) Register(p types.SecretProvider) {
	r.providers[p.Name()] = p
}

// Get returns a provider by name.
func (r *Registry) Get(name string) (types.SecretProvider, error) {
	p, ok := r.providers[name]
	if !ok {
		available := make([]string, 0, len(r.providers))
		for k := range r.providers {
			available = append(available, k)
		}
		return nil, fmt.Errorf("unknown secret provider %q, available: %v", name, available)
	}
	return p, nil
}

// LoadSecrets loads secrets from the configured provider.
func (r *Registry) LoadSecrets(cfg types.SecretsConfig, keys []string) (map[string]string, error) {
	provider, err := r.Get(cfg.Provider)
	if err != nil {
		return nil, err
	}

	if !provider.Available() {
		return nil, fmt.Errorf("secret provider %q is not available on this system", cfg.Provider)
	}

	return provider.Load(keys)
}
