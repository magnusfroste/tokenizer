// Package openai contains the minimal OpenAI-compatible request and response
// shapes that the router accepts on its public API and emits to clients.
package openai

type ChatRequest struct {
	Model               string         `json:"model"`
	Messages            []Message      `json:"messages"`
	Temperature         *float64       `json:"temperature,omitempty"`
	MaxTokens           *int           `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int           `json:"max_completion_tokens,omitempty"`
	Stream              bool           `json:"stream,omitempty"`
	StreamOptions       *StreamOptions `json:"stream_options,omitempty"`
	Tools               []any          `json:"tools,omitempty"`
	ResponseFormat      any            `json:"response_format,omitempty"`
	Metadata            map[string]any `json:"metadata,omitempty"`
}

// StreamOptions mirrors the OpenAI/OpenRouter streaming options. IncludeUsage
// asks the provider to emit a final chunk carrying token usage so the router can
// account real tokens/cost for streamed requests.
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// ModelObject is one entry in an OpenAI-style GET /v1/models response.
type ModelObject struct {
	ID      string `json:"id"`
	Object  string `json:"object"` // always "model"
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelList is the OpenAI-style GET /v1/models response envelope.
type ModelList struct {
	Object string        `json:"object"` // always "list"
	Data   []ModelObject `json:"data"`
}
