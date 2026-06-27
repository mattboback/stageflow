package scannerregistry

import (
	"sort"
	"strings"
)

// Get retrieves a scanner by ID.
func (r *Registry) Get(id string) (*Definition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.scanners[id]

	return def.clone(), ok
}

// Resolve looks up a scanner by ID or alias.
func (r *Registry) Resolve(idOrAlias string) (*Definition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if def, ok := r.scanners[idOrAlias]; ok {
		return def.clone(), true
	}

	if id, aliasFound := r.aliasMap[strings.ToLower(idOrAlias)]; aliasFound {
		if def, defFound := r.scanners[id]; defFound {
			return def.clone(), true
		}
	}

	return nil, false
}

// List returns all registered scanners, sorted by ID.
func (r *Registry) List() []*Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.scanners))
	for id := range r.scanners {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	result := make([]*Definition, 0, len(ids))
	for _, id := range ids {
		result = append(result, r.scanners[id].clone())
	}

	return result
}

// Categories returns a stable list of known categories.
func (r *Registry) Categories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cats := make([]string, 0, len(r.categoryMap))
	for cat := range r.categoryMap {
		cats = append(cats, cat)
	}

	sort.Strings(cats)

	return cats
}

// ListEnabled returns only enabled scanners (sorted by ID).
func (r *Registry) ListEnabled() []*Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.scanners))
	for id, def := range r.scanners {
		if def != nil && def.Enabled {
			ids = append(ids, id)
		}
	}

	sort.Strings(ids)

	result := make([]*Definition, 0, len(ids))
	for _, id := range ids {
		result = append(result, r.scanners[id].clone())
	}

	return result
}

// ListByCategory returns enabled scanners in a specific category.
func (r *Registry) ListByCategory(category string) []*Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids, ok := r.categoryMap[category]
	if !ok {
		return nil
	}

	result := make([]*Definition, 0, len(ids))
	for _, id := range ids {
		if def, found := r.scanners[id]; found && def.Enabled {
			result = append(result, def.clone())
		}
	}

	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })

	return result
}

// GetImage returns the container image for a scanner.
func (r *Registry) GetImage(id string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if def, ok := r.scanners[id]; ok && def.Image != "" {
		return def.Image
	}

	return r.defaultImage
}

// Count returns the number of registered scanners.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.scanners)
}

// CountEnabled returns the number of enabled scanners.
func (r *Registry) CountEnabled() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0

	for _, def := range r.scanners {
		if def.Enabled {
			count++
		}
	}

	return count
}
