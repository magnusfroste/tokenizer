// Package contextproc runs optional context processors between policy and
// provider adapter translation.
package contextproc

import (
	"context"

	"github.com/magnusfroste/tokenizer/internal/provider"
	"github.com/magnusfroste/tokenizer/internal/router"
)

type Processor interface {
	Name() string
	Process(ctx context.Context, req *provider.NormalizedModelRequest, job *router.JobDescriptor) (Result, error)
}

type Result struct {
	TokensSaved   int
	AppliedRules  []string
	SkippedReason string
}
