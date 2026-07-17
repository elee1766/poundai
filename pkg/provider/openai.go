package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/elee1766/zsh_poundai/pkg/config"
)

// Default endpoints per OpenAI-compatible provider name.
var openAIBaseURLs = map[string]string{
	"openai":     "https://api.openai.com/v1",
	"groq":       "https://api.groq.com/openai/v1",
	"mistral":    "https://api.mistral.ai/v1",
	"openrouter": "https://openrouter.ai/api/v1",
}

var openAIDefaultModels = map[string]string{
	"openai":  "gpt-4o-mini",
	"groq":    "llama-3.3-70b-versatile",
	"mistral": "codestral-latest",
}

// openAI speaks the OpenAI chat completions API. It also covers Groq,
// Mistral, OpenRouter, Ollama's /v1 endpoint, llama.cpp, vLLM, LM Studio, and
// anything else OpenAI-compatible via base_url.
type openAI struct {
	svc     config.Service
	baseURL string
	model   string
	apiKey  string
}

func newOpenAI(svc config.Service) (*openAI, error) {
	baseURL := svc.BaseURL
	if baseURL == "" {
		baseURL = openAIBaseURLs[svc.Provider]
	}
	if baseURL == "" {
		return nil, fmt.Errorf("provider %q: base_url is required", svc.Provider)
	}
	model := svc.Model
	if model == "" {
		model = openAIDefaultModels[svc.Provider]
	}
	if model == "" {
		return nil, fmt.Errorf("provider %q: model is required", svc.Provider)
	}
	// Require an API key for cloud providers; local endpoints (custom base_url) may not need one.
	apiKey := svc.ResolveAPIKey()
	if apiKey == "" && svc.BaseURL == "" {
		return nil, fmt.Errorf("provider %q: api_key (or api_key_env) is required", svc.Provider)
	}
	return &openAI{svc: svc, baseURL: strings.TrimRight(baseURL, "/"), model: model, apiKey: apiKey}, nil
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *openAI) Complete(ctx context.Context, messages []Message) (string, error) {
	headers := map[string]string{}
	if p.apiKey != "" {
		headers["Authorization"] = "Bearer " + p.apiKey
	}
	if p.svc.Organization != "" {
		headers["OpenAI-Organization"] = p.svc.Organization
	}
	for k, v := range p.svc.ExtraHeaders {
		if _, reserved := headers[k]; reserved {
			continue // don't let ExtraHeaders overwrite auth headers
		}
		headers[k] = v
	}
	req := chatRequest{
		Model:       p.model,
		Messages:    messages,
		Temperature: p.svc.Temperature,
		MaxTokens:   p.svc.MaxTokens,
	}
	var resp chatResponse
	if err := postJSON(ctx, httpClient(p.svc), p.baseURL+"/chat/completions", headers, req, &resp); err != nil {
		return "", err
	}
	if resp.Error != nil {
		return "", fmt.Errorf("api error: %s", resp.Error.Message)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("api returned no choices")
	}
	return resp.Choices[0].Message.Content, nil
}
