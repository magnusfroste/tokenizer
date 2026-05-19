// Package registry owns the static in-memory model and provider registry used
// by routing fast paths.
package registry

import "time"

type Tier string

const (
	TierCheap    Tier = "cheap"
	TierBalanced Tier = "balanced"
	TierPremium  Tier = "premium"
)

type ProviderStatus string

const (
	ProviderStatusActive   ProviderStatus = "active"
	ProviderStatusDisabled ProviderStatus = "disabled"
)

type Capabilities struct {
	Chat        bool
	Streaming   bool
	ToolCalls   bool
	JSONSchema  bool
	Vision      bool
	LongContext bool
}

func (c Capabilities) Satisfies(required Capabilities) bool {
	if required.Chat && !c.Chat {
		return false
	}
	if required.Streaming && !c.Streaming {
		return false
	}
	if required.ToolCalls && !c.ToolCalls {
		return false
	}
	if required.JSONSchema && !c.JSONSchema {
		return false
	}
	if required.Vision && !c.Vision {
		return false
	}
	if required.LongContext && !c.LongContext {
		return false
	}
	return true
}

func (c Capabilities) Merge(other Capabilities) Capabilities {
	return Capabilities{
		Chat:        c.Chat || other.Chat,
		Streaming:   c.Streaming || other.Streaming,
		ToolCalls:   c.ToolCalls || other.ToolCalls,
		JSONSchema:  c.JSONSchema || other.JSONSchema,
		Vision:      c.Vision || other.Vision,
		LongContext: c.LongContext || other.LongContext,
	}
}

type CostMetadata struct {
	Currency                    string
	InputMicrosPerMillionToken  int64
	OutputMicrosPerMillionToken int64
}

func (c CostMetadata) Available() bool {
	return c.Currency != "" && (c.InputMicrosPerMillionToken > 0 || c.OutputMicrosPerMillionToken > 0)
}

type LatencyMetadata struct {
	P50FirstTokenMS int
	P95FirstTokenMS int
}

type Provider struct {
	ID            string
	Name          string
	Status        ProviderStatus
	BaseURL       string
	AuthSecretRef string
}

type Model struct {
	ID                  string
	ProviderID          string
	ProviderModelID     string
	Tier                Tier
	Capabilities        Capabilities
	Cost                CostMetadata
	ContextWindowTokens int
	Enabled             bool
	Latency             LatencyMetadata
	QualityScores       map[string]float64
	Strengths           []string
	Weaknesses          []string
}

type Definition struct {
	RegistryVersion string
	CreatedAt       time.Time
	Providers       []Provider
	Models          []Model
}
