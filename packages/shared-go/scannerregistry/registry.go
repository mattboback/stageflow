package scannerregistry

import (
	"sync"
)

// Registry tracks scanners, aliases, and categories for module resolution.
type Registry struct {
	mu           sync.RWMutex
	scanners     map[string]*Definition
	aliasMap     map[string]string   // alias -> scanner ID
	categoryMap  map[string][]string // category -> scanner IDs
	defaultImage string
}

// NewRegistry initializes an empty registry with a default image fallback.
func NewRegistry(defaultImage string) *Registry {
	return &Registry{
		scanners:     make(map[string]*Definition),
		aliasMap:     make(map[string]string),
		categoryMap:  make(map[string][]string),
		defaultImage: defaultImage,
	}
}
