// Package provider contains the provider-adapter interface and the in-tree
// adapters. In sprint 1 only the mock adapter is implemented; real provider
// adapters (OpenAI, Anthropic, ...) arrive in EPIC-06.
package provider

import (
	"context"
	"errors"

	"github.com/magnusfroste/tokenizer/internal/openai"
)

type Adapter interface {
	Name() string
	Complete(ctx context.Context, req *openai.ChatRequest) (*openai.ChatResponse, error)
}

var (
	ErrProviderTimeout   = errors.New("provider_timeout")
	ErrProviderRateLimit = errors.New("provider_rate_limit")
	ErrProvider5xx       = errors.New("provider_5xx")
	ErrProviderBadResp   = errors.New("provider_bad_response")
)
