package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/elee1766/poundai/pkg/config"
)

const anthropicVersion = "2023-06-01"

// anthropic speaks the Anthropic Messages API.
type anthropic struct {
	svc     config.Service
	baseURL string
	model   string
	apiKey  string
}

func newAnthropic(svc config.Service) (*anthropic, error) {
	key := svc.ResolveAPIKey()
	if key == "" {
		return nil, fmt.Errorf("provider \"anthropic\": api_key (or api_key_env) is required")
	}
	baseURL := svc.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	model := svc.Model
	if model == "" {
		model = "claude-3-5-haiku-latest"
	}
	return &anthropic{svc: svc, baseURL: strings.TrimRight(baseURL, "/"), model: model, apiKey: key}, nil
}

type anthropicRequest struct {
	Model       string    `json:"model"`
	System      string    `json:"system,omitempty"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature *float64  `json:"temperature,omitempty"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *anthropic) Complete(ctx context.Context, messages []Message) (string, error) {
	maxTokens := p.svc.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1000
	}
	headers := map[string]string{
		"x-api-key":         p.apiKey,
		"anthropic-version": anthropicVersion,
	}
	mergeHeaders(headers, p.svc.ExtraHeaders)
	system, rest := splitSystem(messages)
	req := anthropicRequest{
		Model:       p.model,
		System:      system,
		Messages:    rest,
		MaxTokens:   maxTokens,
		Temperature: p.svc.Temperature,
	}
	var resp anthropicResponse
	if err := postJSON(ctx, httpClient(p.svc), p.baseURL+"/messages", headers, req, &resp); err != nil {
		return "", err
	}
	if resp.Error != nil {
		return "", fmt.Errorf("anthropic error: %s", resp.Error.Message)
	}
	var sb strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	if sb.Len() == 0 {
		return "", fmt.Errorf("anthropic returned no text content")
	}
	return sb.String(), nil
}
