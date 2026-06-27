package scannerregistry

import (
	"errors"
	"fmt"
	"strings"
)

// Register adds or updates a scanner definition.
func (r *Registry) Register(def *Definition) error {
	if def.ID == "" {
		return errors.New("scanner ID is required")
	}

	if def.Name == "" {
		return errors.New("scanner name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, alias := range def.Aliases {
		if existingID, ok := r.aliasMap[strings.ToLower(alias)]; ok && existingID != def.ID {
			return fmt.Errorf("alias '%s' already used by scanner '%s'", alias, existingID)
		}
	}

	if existing, ok := r.scanners[def.ID]; ok {
		for _, cat := range existing.Categories {
			r.removeCategoryMapping(cat, def.ID)
		}

		for _, alias := range existing.Aliases {
			delete(r.aliasMap, strings.ToLower(alias))
		}
	}

	r.scanners[def.ID] = def.clone()

	// Treat the scanner ID as an implicit alias.
	r.aliasMap[strings.ToLower(def.ID)] = def.ID
	for _, alias := range def.Aliases {
		r.aliasMap[strings.ToLower(alias)] = def.ID
	}

	for _, cat := range def.Categories {
		r.categoryMap[cat] = append(r.categoryMap[cat], def.ID)
	}

	return nil
}

// Unregister removes a scanner from the registry.
func (r *Registry) Unregister(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	def, ok := r.scanners[id]
	if !ok {
		return false
	}

	for _, cat := range def.Categories {
		r.removeCategoryMapping(cat, id)
	}

	delete(r.aliasMap, strings.ToLower(id))

	for _, alias := range def.Aliases {
		delete(r.aliasMap, strings.ToLower(alias))
	}

	delete(r.scanners, id)

	return true
}

func (r *Registry) removeCategoryMapping(category, id string) {
	ids := r.categoryMap[category]
	for i, existing := range ids {
		if existing == id {
			r.categoryMap[category] = append(ids[:i], ids[i+1:]...)

			break
		}
	}

	if len(r.categoryMap[category]) == 0 {
		delete(r.categoryMap, category)
	}
}
