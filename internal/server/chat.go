package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/magnusfroste/tokenix/internal/openai"
	"github.com/magnusfroste/tokenix/internal/provider"
)

func ChatCompletionsHandler(p provider.Adapter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req openai.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
			return
		}
		if len(req.Messages) == 0 {
			writeError(w, http.StatusBadRequest, "invalid_request_error", "messages cannot be empty")
			return
		}
		if req.Stream {
			writeError(w, http.StatusNotImplemented, "not_implemented", "streaming is not supported in sprint 1")
			return
		}

		resp, err := p.Complete(r.Context(), &req)
		if err != nil {
			status, code := mapProviderError(err)
			writeError(w, status, code, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Router-Selected-Model", resp.Model)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func mapProviderError(err error) (status int, code string) {
	switch {
	case errors.Is(err, provider.ErrProviderTimeout):
		return http.StatusGatewayTimeout, "provider_timeout"
	case errors.Is(err, provider.ErrProviderRateLimit):
		return http.StatusTooManyRequests, "provider_rate_limit"
	case errors.Is(err, provider.ErrProvider5xx):
		return http.StatusBadGateway, "provider_5xx"
	case errors.Is(err, provider.ErrProviderBadResp):
		return http.StatusBadGateway, "provider_bad_response"
	default:
		return http.StatusBadGateway, "provider_error"
	}
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(openai.ErrorEnvelope{
		Error: openai.ErrorBody{Message: msg, Type: code, Code: code},
	})
}
