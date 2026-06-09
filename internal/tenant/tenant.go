// Package tenant carries the authenticated principal across the request
// lifecycle. Populated by the auth middleware and consumed by downstream
// handlers (policy, routing, logging).
package tenant

import "context"

type Tenant struct {
	ID      string
	Project string
	KeyID   string
	// Scopes lists the capabilities the API key grants. An empty set means the
	// key is unrestricted (legacy keys); the wildcard "*" grants everything.
	Scopes []string
}

// ScopeWildcard grants every scope when present in a tenant's scope set.
const ScopeWildcard = "*"

// HasScope reports whether the tenant's key grants the required scope. A nil
// tenant grants nothing; an empty scope set is treated as unrestricted so keys
// provisioned without explicit scopes keep working.
func (t *Tenant) HasScope(required string) bool {
	if t == nil {
		return false
	}
	if len(t.Scopes) == 0 {
		return true
	}
	for _, s := range t.Scopes {
		if s == required || s == ScopeWildcard {
			return true
		}
	}
	return false
}

type ctxKey struct{}

func WithTenant(ctx context.Context, t *Tenant) context.Context {
	return context.WithValue(ctx, ctxKey{}, t)
}

func FromContext(ctx context.Context) (*Tenant, bool) {
	t, ok := ctx.Value(ctxKey{}).(*Tenant)
	return t, ok
}
