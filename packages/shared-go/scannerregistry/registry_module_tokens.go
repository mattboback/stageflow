package scannerregistry

import "fmt"

type tokenKind int

const (
	tokenKindUnknown tokenKind = iota
	tokenKindScanner
	tokenKindCategory
)

type tokenResolution struct {
	kind       tokenKind
	ids        []string
	disabledID string
	ok         bool
}

func (r *Registry) resolveToken(token string) tokenResolution {
	if token == "" {
		return tokenResolution{}
	}

	if id, ok := r.aliasMap[token]; ok {
		def := r.scanners[id]
		if def == nil || !def.Enabled {
			return tokenResolution{kind: tokenKindScanner, disabledID: id, ok: true}
		}

		return tokenResolution{kind: tokenKindScanner, ids: []string{id}, ok: true}
	}

	if ids, ok := r.categoryMap[token]; ok {
		enabled := make([]string, 0, len(ids))
		for _, id := range ids {
			def := r.scanners[id]
			if def == nil || !def.Enabled {
				continue
			}

			enabled = append(enabled, id)
		}

		if len(enabled) == 0 {
			return tokenResolution{kind: tokenKindCategory, ok: true}
		}

		return tokenResolution{kind: tokenKindCategory, ids: enabled, ok: true}
	}

	return tokenResolution{}
}

func (r *Registry) resolveStrictToken(token string) ([]string, error) {
	res := r.resolveToken(token)
	if !res.ok {
		return nil, fmt.Errorf("unknown scanner module '%s'", token)
	}

	if res.kind == tokenKindScanner && len(res.ids) == 0 {
		return nil, fmt.Errorf("scanner '%s' is disabled", res.disabledID)
	}

	if res.kind == tokenKindCategory && len(res.ids) == 0 {
		return nil, fmt.Errorf("no enabled scanners in category '%s'", token)
	}

	return res.ids, nil
}
