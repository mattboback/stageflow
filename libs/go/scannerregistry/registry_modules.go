package scannerregistry

import (
	"sort"
	"strings"
)

// enabledIDsLocked returns every enabled scanner ID, sorted: the default set when
// a request names no modules. The caller holds r.mu.
func (r *Registry) enabledIDsLocked() []string {
	var ids []string

	for id, def := range r.scanners {
		if def.Enabled {
			ids = append(ids, id)
		}
	}

	sort.Strings(ids)

	return ids
}

// ResolveModules converts module names to scanner IDs; no modules means every
// enabled scanner. Tokens may be scanner IDs, aliases, or category names.
// Unknown tokens are passed through.
func (r *Registry) ResolveModules(modules []string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(modules) == 0 {
		return r.enabledIDsLocked()
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

// ResolveModulesStrict converts module names to enabled scanner IDs; no modules
// means every enabled scanner. Tokens may be scanner IDs, aliases, or category
// names.
func (r *Registry) ResolveModulesStrict(modules []string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(modules) == 0 {
		if ids := r.enabledIDsLocked(); len(ids) > 0 {
			return ids, nil
		}

		return nil, &ModuleError{message: "no scanners selected"}
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
		return nil, &ModuleError{message: "no scanners selected"}
	}

	sort.Strings(result)

	return result, nil
}
