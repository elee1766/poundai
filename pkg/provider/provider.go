// Package provider implements LLM backends for completion generation.
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/elee1766/poundai/pkg/config"
)

// DefaultTimeout is used when a service does not configure one.
const DefaultTimeout = 30 * time.Second

// Message is a single chat message.
type Message struct {
	Role    string `json:"role"` // "system", "user", or "assistant"
	Content string `json:"content"`
}

// Provider generates a completion for the given conversation.
type Provider interface {
	Complete(ctx context.Context, messages []Message) (string, error)
}

// splitSystem separates system messages (concatenated) from the rest of the
// conversation, for APIs that take the system prompt out-of-band.
func splitSystem(messages []Message) (system string, rest []Message) {
	var sys []string
	for _, m := range messages {
		if m.Role == "system" {
			sys = append(sys, m.Content)
		} else {
			rest = append(rest, m)
		}
	}
	return strings.Join(sys, "\n\n"), rest
}

// New builds a Provider from a service profile.
func New(svc config.Service) (Provider, error) {
	switch svc.Provider {
	case "openai", "groq", "mistral", "openrouter":
		return newOpenAI(svc)
	case "ollama":
		return newOllama(svc)
	case "gemini":
		return newGemini(svc)
	case "anthropic":
		return newAnthropic(svc)
	case "bedrock":
		return newBedrock(svc)
	default:
		return nil, fmt.Errorf("unknown provider %q (want openai, groq, mistral, openrouter, ollama, gemini, anthropic, or bedrock)", svc.Provider)
	}
}

func mergeHeaders(headers, extra map[string]string) {
	for key, value := range extra {
		reserved := false
		for existing := range headers {
			if strings.EqualFold(key, existing) {
				reserved = true
				break
			}
		}
		if !reserved {
			headers[key] = value
		}
	}
}

// httpClient returns an http.Client with the service timeout applied.
func httpClient(svc config.Service) *http.Client {
	return &http.Client{Timeout: svc.Timeout.Std(DefaultTimeout)}
}

// postJSON marshals body, POSTs it to url with the given headers, and decodes
// the JSON response into out. Non-2xx responses are returned as errors that
// include (a truncated portion of) the response body.
func postJSON(ctx context.Context, client *http.Client, url string, headers map[string]string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s: HTTP %d: %s", url, resp.StatusCode, truncate(string(data), 512))
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
