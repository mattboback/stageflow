package scannerregistry

import (
	"errors"
	"sort"
	"strings"
)

const defaultModuleAxe = "axe"

// ResolveModules converts module names to scanner IDs.
// Tokens may be scanner IDs, aliases, or category names. Unknown tokens are passed through.
func (r *Registry) ResolveModules(modules []string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(modules) == 0 {
		if def, ok := r.scanners[defaultModuleAxe]; ok && def.Enabled {
			return []string{defaultModuleAxe}
		}

		return nil
	}

	seen := make(map[string]bool)
	result := make([]string, 0, len(modules))

	for _, raw := range modules {
		token := strings.ToLower(strings.TrimSpace(raw))
		if token == "" {
			continue
		}

		res := r.resolveToken(token)
		for _, id := range res.ids {
			if !seen[id] {
				seen[id] = true
				result = append(result, id)
			}
		}

		if res.ok {
			continue
		}

		// Pass through unknown module IDs to support custom scanners.
		if !seen[token] {
			seen[token] = true
			result = append(result, token)
		}
	}

	sort.Strings(result)

	return result
}

// ResolveModulesStrict converts module names to enabled scanner IDs.
// Tokens may be scanner IDs, aliases, or category names.
func (r *Registry) ResolveModulesStrict(modules []string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(modules) == 0 {
		if def, ok := r.scanners[defaultModuleAxe]; ok && def.Enabled {
			return []string{defaultModuleAxe}, nil
		}

		return nil, errors.New("no scanners selected")
	}

	seen := make(map[string]bool)
	result := make([]string, 0, len(modules))

	for _, raw := range modules {
		token := strings.ToLower(strings.TrimSpace(raw))
		if token == "" {
			continue
		}

		ids, err := r.resolveStrictToken(token)
		if err != nil {
			return nil, err
		}

		for _, id := range ids {
			if !seen[id] {
				seen[id] = true
				result = append(result, id)
			}
		}
	}

	if len(result) == 0 {
		return nil, errors.New("no scanners selected")
	}

	sort.Strings(result)

	return result, nil
}
