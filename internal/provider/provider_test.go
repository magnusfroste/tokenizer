package provider

import (
	"testing"

	"github.com/magnusfroste/tokenizer/internal/openai"
)

func TestNormalizedModelRequestCloneIsolatesNestedToolsAndMetadata(t *testing.T) {
	temperature := 0.2
	maxTokens := 10
	req := NormalizeChatRequest(&openai.ChatRequest{
		Model:       "auto",
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
		Tools: []any{
			map[string]any{
				"function": map[string]any{
					"name": "search",
				},
			},
		},
		Metadata: map[string]any{
			"trace": map[string]any{
				"tenant": "tn_1",
			},
			"tags": []any{"alpha"},
		},
	})

	clone := req.Clone()
	*clone.Temperature = 0.8
	*clone.MaxTokens = 20
	clone.Tools[0].(map[string]any)["function"].(map[string]any)["name"] = "mutated"
	clone.Metadata["trace"].(map[string]any)["tenant"] = "tn_2"
	clone.Metadata["tags"].([]any)[0] = "beta"

	if *req.Temperature != 0.2 {
		t.Fatalf("temperature pointer was shared, got %v", *req.Temperature)
	}
	if *req.MaxTokens != 10 {
		t.Fatalf("max tokens pointer was shared, got %v", *req.MaxTokens)
	}
	gotToolName := req.Tools[0].(map[string]any)["function"].(map[string]any)["name"]
	if gotToolName != "search" {
		t.Fatalf("nested tool map was shared, got %v", gotToolName)
	}
	gotTenant := req.Metadata["trace"].(map[string]any)["tenant"]
	if gotTenant != "tn_1" {
		t.Fatalf("nested metadata map was shared, got %v", gotTenant)
	}
	gotTag := req.Metadata["tags"].([]any)[0]
	if gotTag != "alpha" {
		t.Fatalf("nested metadata slice was shared, got %v", gotTag)
	}
}

func TestNormalizeChatRequestIsolatesSourceNestedToolsAndMetadata(t *testing.T) {
	req := &openai.ChatRequest{
		Model: "auto",
		Tools: []any{
			map[string]any{"function": map[string]any{"name": "search"}},
		},
		Metadata: map[string]any{
			"trace": map[string]any{"tenant": "tn_1"},
		},
	}

	normalized := NormalizeChatRequest(req)
	normalized.Tools[0].(map[string]any)["function"].(map[string]any)["name"] = "mutated"
	normalized.Metadata["trace"].(map[string]any)["tenant"] = "tn_2"

	gotToolName := req.Tools[0].(map[string]any)["function"].(map[string]any)["name"]
	if gotToolName != "search" {
		t.Fatalf("source nested tool map was shared, got %v", gotToolName)
	}
	gotTenant := req.Metadata["trace"].(map[string]any)["tenant"]
	if gotTenant != "tn_1" {
		t.Fatalf("source nested metadata map was shared, got %v", gotTenant)
	}
}
