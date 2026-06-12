package policy

import (
	"fmt"
	"os"

	"github.com/magnusfroste/tokenizer/internal/registry"
)

const defaultRuntimePolicy = `
version: pv_runtime_2026_06_12
metadata:
  owner: platform
  description: Built-in runtime policy
settings:
  default_model_profile: balanced
  conservative_unknowns: true
  max_router_overhead_ms: 100
  default_timeout_ms: 30000
  default_retention: standard
rules:
  - id: block_disabled
    when:
      router_mode: disabled
    route:
      block:
        code: router_disabled
        reason: disabled
`

// NewDefaultRuntimeCache builds the built-in compiled policy cache used by the
// router when no external policy loader exists yet.
func NewDefaultRuntimeCache(snapshot *registry.Snapshot) (*Cache, error) {
	return NewRuntimeCache(snapshot, "")
}

// NewRuntimeCache builds the runtime compiled policy cache. When path is empty
// it uses the built-in default policy; otherwise it parses the provided policy
// file as the default runtime scope. Tenant/project matching remains expressed
// inside policy rules, so request-path lookup stays in-memory.
func NewRuntimeCache(snapshot *registry.Snapshot, path string) (*Cache, error) {
	var (
		data []byte
		err  error
	)
	if path == "" {
		data = []byte(defaultRuntimePolicy)
	} else {
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("policy: read runtime policy %q: %w", path, err)
		}
	}
	parsed, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("policy: parse runtime policy: %w", err)
	}
	cache, err := NewCache([]Source{{Scope: Scope{}, Policy: parsed, Registry: snapshot}})
	if err != nil {
		return nil, fmt.Errorf("policy: compile runtime policy: %w", err)
	}
	return cache, nil
}
