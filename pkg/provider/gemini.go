package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/elee1766/poundai/pkg/config"
)

// gemini speaks the Google Generative Language REST API.
type gemini struct {
	svc     config.Service
	baseURL string
	model   string
	apiKey  string
}

func newGemini(svc config.Service) (*gemini, error) {
	key := svc.ResolveAPIKey()
	if key == "" {
		return nil, fmt.Errorf("provider \"gemini\": api_key (or api_key_env) is required")
	}
	baseURL := svc.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	model := svc.Model
	if model == "" {
		model = "gemini-2.0-flash"
	}
	return &gemini{svc: svc, baseURL: strings.TrimRight(baseURL, "/"), model: model, apiKey: key}, nil
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  map[string]any  `json:"generationConfig,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *gemini) Complete(ctx context.Context, messages []Message) (string, error) {
	system, rest := splitSystem(messages)
	var contents []geminiContent
	for _, m := range rest {
		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{Role: role, Parts: []geminiPart{{Text: m.Content}}})
	}
	req := geminiRequest{
		Contents: contents,
	}
	if system != "" {
		req.SystemInstruction = &geminiContent{Parts: []geminiPart{{Text: system}}}
	}
	gen := map[string]any{}
	if p.svc.Temperature != nil {
		gen["temperature"] = *p.svc.Temperature
	}
	if p.svc.MaxTokens > 0 {
		gen["maxOutputTokens"] = p.svc.MaxTokens
	}
	if len(gen) > 0 {
		req.GenerationConfig = gen
	}
	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, url.PathEscape(p.model), url.QueryEscape(p.apiKey))
	// Use a redacted URL for error messages to avoid leaking the API key.
	redactedEndpoint := fmt.Sprintf("%s/models/%s:generateContent?key=REDACTED", p.baseURL, url.PathEscape(p.model))
	headers := map[string]string{}
	for k, v := range p.svc.ExtraHeaders {
		headers[k] = v
	}
	var resp geminiResponse
	if err := postJSON(ctx, httpClient(p.svc), endpoint, headers, req, &resp); err != nil {
		// Replace the full URL (which contains the API key) with the redacted version.
		return "", fmt.Errorf("%s", strings.Replace(err.Error(), endpoint, redactedEndpoint, 1))
	}
	if resp.Error != nil {
		return "", fmt.Errorf("gemini error: %s", resp.Error.Message)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned no candidates")
	}
	var sb strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		sb.WriteString(part.Text)
	}
	return sb.String(), nil
}
