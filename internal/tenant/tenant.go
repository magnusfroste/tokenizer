// Package tenant carries the authenticated principal across the request
// lifecycle. Populated by the auth middleware and consumed by downstream
// handlers (policy, routing, logging).
package tenant

import "context"

type Tenant struct {
	ID      string
	Project string
	KeyID   string
}

type ctxKey struct{}

func WithTenant(ctx context.Context, t *Tenant) context.Context {
	return context.WithValue(ctx, ctxKey{}, t)
}

func FromContext(ctx context.Context) (*Tenant, bool) {
	t, ok := ctx.Value(ctxKey{}).(*Tenant)
	return t, ok
}
