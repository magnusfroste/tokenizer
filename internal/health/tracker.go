// Package health implements an in-memory rolling-window provider health tracker.
// It satisfies the engine.HealthSnapshot interface so the routing engine can
// read health scores without any I/O on the hot path.
package health

import (
	"sync"
)

// windowSize is the number of recent attempts used for error-rate computation.
const windowSize = 100

// minAttempts is the minimum number of recorded attempts before the tracker
// starts reducing the score below 1.0. Fewer attempts → optimistic score.
const minAttempts = 5

// Tracker is a concurrent, in-memory rolling-window health tracker.
// It records successes and failures per provider and exposes a [0, 1] health
// score used by the routing engine to penalise degraded providers.
type Tracker struct {
	mu      sync.RWMutex
	windows map[string]*providerWindow
}

type providerWindow struct {
	buf   [windowSize]bool // true = success
	head  int
	total int // total attempts stored (capped at windowSize)
}

// New returns a ready-to-use Tracker.
func New() *Tracker {
	return &Tracker{windows: make(map[string]*providerWindow)}
}

// RecordSuccess records a successful call to the given provider.
func (t *Tracker) RecordSuccess(providerID string) {
	t.record(providerID, true)
}

// RecordFailure records a failed call to the given provider.
func (t *Tracker) RecordFailure(providerID string) {
	t.record(providerID, false)
}

func (t *Tracker) record(providerID string, success bool) {
	t.mu.Lock()
	w, ok := t.windows[providerID]
	if !ok {
		w = &providerWindow{}
		t.windows[providerID] = w
	}
	w.buf[w.head%windowSize] = success
	w.head++
	if w.total < windowSize {
		w.total++
	}
	t.mu.Unlock()
}

// ProviderHealth returns a health score in [0.0, 1.0] for the given provider.
// If fewer than minAttempts have been recorded the score is 1.0 (optimistic).
func (t *Tracker) ProviderHealth(providerID string) float64 {
	t.mu.RLock()
	w, ok := t.windows[providerID]
	if !ok || w.total < minAttempts {
		t.mu.RUnlock()
		return 1.0
	}
	n := w.total
	buf := w.buf
	t.mu.RUnlock()

	successes := 0
	for i := 0; i < n; i++ {
		if buf[i] {
			successes++
		}
	}
	return float64(successes) / float64(n)
}

// Providers returns a snapshot of all tracked provider IDs and their scores.
func (t *Tracker) Providers() map[string]float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make(map[string]float64, len(t.windows))
	for id, w := range t.windows {
		if w.total < minAttempts {
			out[id] = 1.0
			continue
		}
		successes := 0
		for i := 0; i < w.total; i++ {
			if w.buf[i] {
				successes++
			}
		}
		out[id] = float64(successes) / float64(w.total)
	}
	return out
}
