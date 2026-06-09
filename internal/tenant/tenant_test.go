package tenant

import "testing"

func TestHasScope(t *testing.T) {
	cases := []struct {
		name     string
		scopes   []string
		required string
		want     bool
	}{
		{"empty set is unrestricted", nil, "chat:completions", true},
		{"wildcard grants all", []string{"*"}, "router:decision", true},
		{"explicit member", []string{"chat:completions", "router:decision"}, "router:decision", true},
		{"explicit non-member", []string{"chat:completions"}, "router:decision", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tn := &Tenant{ID: "tn", Scopes: c.scopes}
			if got := tn.HasScope(c.required); got != c.want {
				t.Errorf("HasScope(%q) with %v = %v, want %v", c.required, c.scopes, got, c.want)
			}
		})
	}
}

func TestHasScopeNilTenant(t *testing.T) {
	var tn *Tenant
	if tn.HasScope("anything") {
		t.Error("nil tenant should grant no scopes")
	}
}
