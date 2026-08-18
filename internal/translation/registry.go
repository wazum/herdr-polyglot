package translation

import (
	"fmt"
	"strings"
)

// The first provider registered is the default, which keeps the choice of
// service in the composition root rather than in configuration handling.
type Registry struct {
	providers map[string]Provider
	order     []string
}

func NewRegistry(providers ...Provider) *Registry {
	registry := &Registry{providers: make(map[string]Provider, len(providers))}
	for _, provider := range providers {
		name := provider.Name()
		if _, known := registry.providers[name]; known {
			continue
		}
		registry.providers[name] = provider
		registry.order = append(registry.order, name)
	}
	return registry
}

func (r *Registry) Default() string {
	if len(r.order) == 0 {
		return ""
	}
	return r.order[0]
}

func (r *Registry) Names() []string {
	return append([]string(nil), r.order...)
}

func (r *Registry) Translator(name string, options Options) (Translator, error) {
	if name == "" {
		name = r.Default()
	}

	provider, known := r.providers[name]
	if !known {
		return nil, fmt.Errorf("unknown translation service %q; available: %s",
			name, strings.Join(r.order, ", "))
	}

	translator, err := provider.New(options)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return translator, nil
}
