package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/elee1766/poundai/pkg/config"
)

// ollama speaks Ollama's native /api/chat endpoint. No API key required.
type ollama struct {
	svc     config.Service
	baseURL string
	model   string
}

func newOllama(svc config.Service) (*ollama, error) {
	if svc.Model == "" {
		return nil, fmt.Errorf("provider \"ollama\": model is required")
	}
	baseURL := svc.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &ollama{svc: svc, baseURL: strings.TrimRight(baseURL, "/"), model: svc.Model}, nil
}

type ollamaRequest struct {
	Model    string         `json:"model"`
	Messages []Message      `json:"messages"`
	Stream   bool           `json:"stream"`
	Options  map[string]any `json:"options,omitempty"`
}

type ollamaResponse struct {
	Message Message `json:"message"`
	Error   string  `json:"error"`
}

func (p *ollama) Complete(ctx context.Context, messages []Message) (string, error) {
	req := ollamaRequest{
		Model:    p.model,
		Messages: messages,
	}
	opts := map[string]any{}
	if p.svc.Temperature != nil {
		opts["temperature"] = *p.svc.Temperature
	}
	if p.svc.MaxTokens > 0 {
		opts["num_predict"] = p.svc.MaxTokens
	}
	if len(opts) > 0 {
		req.Options = opts
	}
	var resp ollamaResponse
	if err := postJSON(ctx, httpClient(p.svc), p.baseURL+"/api/chat", p.svc.ExtraHeaders, req, &resp); err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", fmt.Errorf("ollama error: %s", resp.Error)
	}
	return resp.Message.Content, nil
}
