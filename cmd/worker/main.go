// Worker is the placeholder for the async event-log queue consumer described
// in 01-architecture/12-observability.md and EPIC-07. It is intentionally a
// no-op in Sprint 1: the architecture says decision/attempt logging is async
// and must not block the request path, so a worker binary exists from day 1
// even though the queue and DB land in Sprint 2 (registry) and Sprint 6
// (observability).
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	logger.Info("worker starting (sprint-1 placeholder; no queue wired yet)")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	logger.Info("worker shutting down")
}
