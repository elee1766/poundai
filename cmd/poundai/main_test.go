package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPluginCommand(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "poundai")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Env = os.Environ()
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	for _, shell := range []string{"zsh", "bash"} {
		want, err := os.ReadFile(filepath.Join("..", "..", "poundai.plugin."+shell))
		if err != nil {
			t.Fatal(err)
		}
		got, err := exec.Command(bin, "plugin", shell).Output()
		if err != nil {
			t.Fatalf("poundai plugin %s: %v", shell, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("poundai plugin %s output differs from source file", shell)
		}
	}

	out, err := exec.Command(bin, "plugin", "fish").CombinedOutput()
	if err == nil || !strings.Contains(string(out), "unsupported shell") {
		t.Errorf("poundai plugin fish = err %v, output %q", err, out)
	}
}

// TestEndToEnd builds the binary and runs it against a mock OpenAI-compatible
// server, exercising config loading, context injection, and cleanup.
func TestEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-to-end test in short mode")
	}

	tmp := t.TempDir()
	bin := filepath.Join(tmp, "poundai")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Env = os.Environ()
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var gotMessages []message
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		gotMessages = req.Messages
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "docker ps -a"}},
			},
		})
	}))
	defer srv.Close()

	// Optimized prompt artifact with one few-shot demo.
	optPath := filepath.Join(tmp, "optimized_prompt.json")
	opt := `{
  "system": "OPTIMIZED_SYSTEM_67890. Output only the shell command.",
  "demos": [{"comment": "show free disk space", "command": "df -h"}]
}`
	if err := os.WriteFile(optPath, []byte(opt), 0o600); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(tmp, "config.yaml")
	cfg := fmt.Sprintf(`
service: test
services:
  test:
    provider: openai
    base_url: %s
    api_key: dummy
    model: test-model
prompt:
  system_extra: "Prefer POSIX-compatible commands."
  optimized_file: %s
context:
  cwd: true
  commands:
    - name: marker
      command: echo CONTEXT_MARKER_12345
`, srv.URL, optPath)
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	buffer := "# list all docker containers"
	cmd := exec.Command(bin, "-config", cfgPath, fmt.Sprint(len(buffer)))
	cmd.Stdin = strings.NewReader(buffer)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("run failed: %v\nstderr: %s", err, ee.Stderr)
		}
		t.Fatal(err)
	}

	// Comment buffer -> completion on the next line.
	if got := string(out); got != "\ndocker ps -a" {
		t.Errorf("output = %q, want %q", got, "\ndocker ps -a")
	}

	// system + demo user/assistant pair + real user = 4 messages.
	if len(gotMessages) != 4 {
		t.Fatalf("got %d messages, want 4: %+v", len(gotMessages), gotMessages)
	}
	gotSystem := gotMessages[0].Content
	// Optimized system prompt is the base; system_extra and context appended.
	for _, want := range []string{
		"OPTIMIZED_SYSTEM_67890",
		"Prefer POSIX-compatible commands.",
		"CONTEXT_MARKER_12345",
		"<cwd>",
	} {
		if !strings.Contains(gotSystem, want) {
			t.Errorf("system prompt missing %q:\n%s", want, gotSystem)
		}
	}
	// Demo pair formatted like a real request.
	demoWant := "#!/bin/zsh\n\nComplete the shell input at the <poundai-cursor/> marker. Return only the text to insert at the marker.\n\n# show free disk space<poundai-cursor/>"
	if gotMessages[1].Role != "user" || gotMessages[1].Content != demoWant {
		t.Errorf("demo user message = %+v", gotMessages[1])
	}
	if gotMessages[2].Role != "assistant" || gotMessages[2].Content != "df -h" {
		t.Errorf("demo assistant message = %+v", gotMessages[2])
	}
	userWant := "#!/bin/zsh\n\nComplete the shell input at the <poundai-cursor/> marker. Return only the text to insert at the marker.\n\n" + buffer + "<poundai-cursor/>"
	if gotMessages[3].Content != userWant {
		t.Errorf("user message = %q", gotMessages[3].Content)
	}
}

// TestEndToEndDefaultPrompt verifies that with no prompt configuration the
// built-in minimal system prompt is used, with no few-shot demos.
func TestEndToEndDefaultPrompt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-to-end test in short mode")
	}

	tmp := t.TempDir()
	bin := filepath.Join(tmp, "poundai")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Env = os.Environ()
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var gotMessages []message
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		gotMessages = req.Messages
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "wc -l access.log"}},
			},
		})
	}))
	defer srv.Close()

	cfgPath := filepath.Join(tmp, "config.yaml")
	cfg := fmt.Sprintf("services:\n  test:\n    provider: openai\n    base_url: %s\n    api_key: dummy\n    model: m\n", srv.URL)
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	buffer := "# count lines in access.log"
	cmd := exec.Command(bin, "-config", cfgPath, "-no-context", fmt.Sprint(len(buffer)))
	cmd.Stdin = strings.NewReader(buffer)
	if _, err := cmd.Output(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("run failed: %v\nstderr: %s", err, ee.Stderr)
		}
		t.Fatal(err)
	}

	// Built-in default prompt: system + user only, no demos.
	if len(gotMessages) != 2 {
		t.Fatalf("got %d messages, want 2 (default prompt has no demos): %+v", len(gotMessages), gotMessages)
	}
	if !strings.Contains(gotMessages[0].Content, "Convert a natural language comment describing a shell task") {
		t.Errorf("system prompt is not the built-in default:\n%.200s", gotMessages[0].Content)
	}
}
