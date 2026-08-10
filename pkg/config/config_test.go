package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/adrg/xdg"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad(t *testing.T) {
	path := writeConfig(t, `
service: local

services:
  local:
    provider: ollama
    model: qwen2.5-coder:7b
    temperature: 0.2
    timeout: 45s
  cloud:
    provider: openai
    api_key_env: OPENAI_API_KEY
    model: gpt-4o-mini

prompt:
  system_extra: "Prefer long flags."

context:
  cwd: true
  os: true
  history: 10
  env: [VIRTUAL_ENV]
  hook: context
  commands:
    - name: git
      command: git status -sb
      timeout: 500ms
      max_bytes: 2048
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := cfg.Active("")
	if err != nil {
		t.Fatal(err)
	}
	if svc.Provider != "ollama" || svc.Model != "qwen2.5-coder:7b" {
		t.Errorf("unexpected active service: %+v", svc)
	}
	if svc.Temperature == nil || *svc.Temperature != 0.2 {
		t.Errorf("temperature = %v", svc.Temperature)
	}
	if svc.Timeout.Std(0) != 45*time.Second {
		t.Errorf("timeout = %v", svc.Timeout.Std(0))
	}
	if cfg.Prompt.SystemExtra != "Prefer long flags." {
		t.Errorf("system_extra = %q", cfg.Prompt.SystemExtra)
	}
	if !cfg.Context.Cwd || cfg.Context.History != 10 {
		t.Errorf("context = %+v", cfg.Context)
	}
	if cfg.Context.Hook != filepath.Join(filepath.Dir(path), "context") {
		t.Errorf("hook = %q", cfg.Context.Hook)
	}
	if len(cfg.Context.Commands) != 1 || cfg.Context.Commands[0].Timeout.Std(0) != 500*time.Millisecond {
		t.Errorf("commands = %+v", cfg.Context.Commands)
	}

	// Explicit override.
	svc, err = cfg.Active("cloud")
	if err != nil {
		t.Fatal(err)
	}
	if svc.Provider != "openai" {
		t.Errorf("override service = %+v", svc)
	}
}

func TestDefaultPath(t *testing.T) {
	oldHome, oldDirs := xdg.ConfigHome, xdg.ConfigDirs
	xdg.ConfigHome, xdg.ConfigDirs = t.TempDir(), nil
	t.Cleanup(func() {
		xdg.ConfigHome, xdg.ConfigDirs = oldHome, oldDirs
	})

	want := filepath.Join(xdg.ConfigHome, "poundai", "config.yml")
	if got := DefaultPath(); got != want {
		t.Errorf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestLoadSingleServiceImplicitSelector(t *testing.T) {
	path := writeConfig(t, `
services:
  only:
    provider: ollama
    model: llama3
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Service != "only" {
		t.Errorf("implicit selector = %q", cfg.Service)
	}
}

func TestLoadErrors(t *testing.T) {
	cases := map[string]string{
		"no services": `service: x`,
		"missing selector with multiple services": `
services:
  a: {provider: ollama, model: m}
  b: {provider: openai, model: m}
`,
		"selector points nowhere": `
service: nope
services:
  a: {provider: ollama, model: m}
`,
		"unknown field": `
service: a
services:
  a: {provider: ollama, model: m}
bogus_key: true
`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(writeConfig(t, content)); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestActiveMissingModel(t *testing.T) {
	path := writeConfig(t, `
service: a
services:
  a:
    provider: ollama
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.Active(""); err == nil {
		t.Error("expected error for missing model")
	}
}

func TestResolveAPIKey(t *testing.T) {
	t.Setenv("POUNDAI_TEST_KEY", "from-env")
	if got := (Service{APIKey: "literal", APIKeyEnv: "POUNDAI_TEST_KEY"}).ResolveAPIKey(); got != "literal" {
		t.Errorf("literal wins: got %q", got)
	}
	if got := (Service{APIKeyEnv: "POUNDAI_TEST_KEY"}).ResolveAPIKey(); got != "from-env" {
		t.Errorf("env fallback: got %q", got)
	}
	if got := (Service{}).ResolveAPIKey(); got != "" {
		t.Errorf("empty: got %q", got)
	}
}
