package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elee1766/zsh_poundai/pkg/config"
)

func TestOpenAIComplete(t *testing.T) {
	var gotAuth, gotPath string
	var gotReq chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Error(err)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "ls -la"}},
			},
		})
	}))
	defer srv.Close()

	temp := 0.3
	p, err := New(config.Service{
		Provider:    "openai",
		BaseURL:     srv.URL,
		APIKey:      "test-key",
		Model:       "test-model",
		Temperature: &temp,
		MaxTokens:   64,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.Complete(context.Background(), []Message{{Role: "system", Content: "system prompt"}, {Role: "user", Content: "user prompt"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "ls -la" {
		t.Errorf("Complete = %q", got)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotPath != "/chat/completions" {
		t.Errorf("path = %q", gotPath)
	}
	if gotReq.Model != "test-model" || len(gotReq.Messages) != 2 ||
		gotReq.Messages[0].Role != "system" || gotReq.Messages[1].Content != "user prompt" {
		t.Errorf("request = %+v", gotReq)
	}
	if gotReq.Temperature == nil || *gotReq.Temperature != 0.3 || gotReq.MaxTokens != 64 {
		t.Errorf("params = %+v", gotReq)
	}
}

func TestOpenAIHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"bad key"}}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	p, err := New(config.Service{Provider: "openai", BaseURL: srv.URL, Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Complete(context.Background(), []Message{{Role: "system", Content: "s"}, {Role: "user", Content: "u"}}); err == nil {
		t.Error("expected error on HTTP 401")
	}
}

func TestOllamaComplete(t *testing.T) {
	var gotReq ollamaRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Error(err)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"role": "assistant", "content": "docker ps"},
		})
	}))
	defer srv.Close()

	p, err := New(config.Service{Provider: "ollama", BaseURL: srv.URL, Model: "llama3"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.Complete(context.Background(), []Message{{Role: "system", Content: "sys"}, {Role: "user", Content: "usr"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "docker ps" {
		t.Errorf("Complete = %q", got)
	}
	if gotReq.Stream {
		t.Error("stream should be false")
	}
}

func TestGeminiComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "gem-key" {
			t.Errorf("key = %q", r.URL.Query().Get("key"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{"parts": []map[string]string{{"text": "kubectl get pods"}}}},
			},
		})
	}))
	defer srv.Close()

	p, err := New(config.Service{Provider: "gemini", BaseURL: srv.URL, APIKey: "gem-key", Model: "gemini-test"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.Complete(context.Background(), []Message{{Role: "system", Content: "sys"}, {Role: "user", Content: "usr"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "kubectl get pods" {
		t.Errorf("Complete = %q", got)
	}
}

func TestAnthropicComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "ant-key" {
			t.Errorf("x-api-key = %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Error("missing anthropic-version header")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{{"type": "text", "text": "rg TODO"}},
		})
	}))
	defer srv.Close()

	p, err := New(config.Service{Provider: "anthropic", BaseURL: srv.URL, APIKey: "ant-key", Model: "claude-test"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.Complete(context.Background(), []Message{{Role: "system", Content: "sys"}, {Role: "user", Content: "usr"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "rg TODO" {
		t.Errorf("Complete = %q", got)
	}
}

func TestNewUnknownProvider(t *testing.T) {
	if _, err := New(config.Service{Provider: "nope"}); err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestNewMissingRequirements(t *testing.T) {
	cases := []struct {
		svc     config.Service
		wantErr bool
	}{
		{config.Service{Provider: "gemini"}, true},                                                   // missing api key
		{config.Service{Provider: "anthropic"}, true},                                                // missing api key
		{config.Service{Provider: "bedrock"}, true},                                                  // missing model
		{config.Service{Provider: "ollama"}, true},                                                   // missing model
		{config.Service{Provider: "openai"}, true},                                                   // missing api key (cloud)
		{config.Service{Provider: "openrouter", Model: "m", APIKey: "k"}, false},                     // has key+model
		{config.Service{Provider: "openai", BaseURL: "http://localhost:8080/v1", Model: "m"}, false}, // local: no key needed
	}
	for _, tc := range cases {
		_, err := New(tc.svc)
		if tc.wantErr && err == nil {
			t.Errorf("provider %q: expected error, got nil", tc.svc.Provider)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("provider %q: unexpected error: %v", tc.svc.Provider, err)
		}
	}
}
