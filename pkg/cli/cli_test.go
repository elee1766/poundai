package cli

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	poundai "github.com/elee1766/poundai"
	"github.com/elee1766/poundai/pkg/config"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}
	tests := []struct {
		input, want string
	}{
		{"~/foo", filepath.Join(home, "foo")},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative", "relative"},
	}
	for _, tt := range tests {
		if got := ExpandPath(tt.input); got != tt.want {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}

	// Test env var expansion.
	t.Setenv("POUNDAI_TEST_DIR", "/tmp/test")
	if got := ExpandPath("$POUNDAI_TEST_DIR/file"); got != "/tmp/test/file" {
		t.Errorf("ExpandPath with env = %q", got)
	}
}

func TestRunInitNonInteractive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "config.yml")
	var out bytes.Buffer
	err := runInit([]string{
		"-config", path,
		"-non-interactive",
		"-provider", "ollama",
		"-model", "qwen-test",
	}, strings.NewReader(""), &out)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := cfg.Active("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Service != "default" || svc.Provider != "ollama" || svc.Model != "qwen-test" {
		t.Errorf("config = %+v", cfg)
	}
	if svc.BaseURL != "http://localhost:11434" {
		t.Errorf("base URL = %q", svc.BaseURL)
	}
	if !cfg.Context.Cwd || !cfg.Context.OS || cfg.Context.History != 10 {
		t.Errorf("context = %+v", cfg.Context)
	}
	if len(cfg.Context.Commands) != 1 || cfg.Context.Commands[0].Name != "git" {
		t.Errorf("commands = %+v", cfg.Context.Commands)
	}
	if !strings.Contains(out.String(), "poundai doctor") {
		t.Errorf("output = %q", out.String())
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("config mode = %o", info.Mode().Perm())
	}
}

func TestRunInitPreservesCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	input := strings.NewReader("OpenAI\nhttps://localhost:8080/v1\nModel-With-Case\n")
	var out bytes.Buffer
	if err := runInit([]string{"-config", path}, input, &out); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := cfg.Active("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Service != "default" {
		t.Errorf("service = %q", cfg.Service)
	}
	if svc.Provider != "openai" {
		t.Errorf("provider = %q", svc.Provider)
	}
	if svc.Model != "Model-With-Case" || svc.BaseURL != "https://localhost:8080/v1" {
		t.Errorf("service = %+v", svc)
	}
}

func TestRunInitHostedProviderSkipsBaseURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	input := strings.NewReader("Groq\nCustomModel\nCUSTOM_GROQ_KEY\n")
	var out bytes.Buffer
	if err := runInit([]string{"-config", path}, input, &out); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := cfg.Active("")
	if err != nil {
		t.Fatal(err)
	}
	if svc.Provider != "groq" || svc.BaseURL != "" || svc.Model != "CustomModel" || svc.APIKeyEnv != "CUSTOM_GROQ_KEY" {
		t.Errorf("service = %+v", svc)
	}
	if strings.Contains(out.String(), "Base URL") {
		t.Errorf("hosted provider prompted for base URL: %q", out.String())
	}
}

func TestRunInitConfiguresHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	input := strings.NewReader("ollama\n\n\n25\n")
	if err := runInit([]string{"-config", path}, input, io.Discard); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Context.History != 25 {
		t.Errorf("history = %d", cfg.Context.History)
	}
}

func TestRunInitRejectsNegativeHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	err := runInit(
		[]string{"-config", path, "-non-interactive", "-history", "-1"},
		strings.NewReader(""),
		io.Discard,
	)
	if err == nil || !strings.Contains(err.Error(), "history count must be non-negative") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunInitConfiguresCommandPresets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := runInit(
		[]string{"-config", path, "-non-interactive", "-commands", "git,files"},
		strings.NewReader(""),
		io.Discard,
	); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Context.Commands) != 2 || cfg.Context.Commands[0].Name != "git" || cfg.Context.Commands[1].Name != "files" {
		t.Errorf("commands = %+v", cfg.Context.Commands)
	}
}

func TestRunInitDisablesCommandPresets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := runInit(
		[]string{"-config", path, "-non-interactive", "-commands", "none"},
		strings.NewReader(""),
		io.Discard,
	); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Context.Commands) != 0 {
		t.Errorf("commands = %+v", cfg.Context.Commands)
	}
}

func TestRunInitRejectsUnknownCommandPreset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	err := runInit(
		[]string{"-config", path, "-non-interactive", "-commands", "unknown"},
		strings.NewReader(""),
		io.Discard,
	)
	if err == nil || !strings.Contains(err.Error(), "unknown context command preset") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunInitRefusesExistingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := runInit(
		[]string{"-config", path, "-non-interactive"},
		strings.NewReader(""),
		io.Discard,
	)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunInitReplacesUntouchedExample(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(poundai.ConfigExample()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runInit(
		[]string{"-config", path, "-non-interactive", "-model", "test-model"},
		strings.NewReader(""),
		io.Discard,
	); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := cfg.Active("")
	if err != nil {
		t.Fatal(err)
	}
	if svc.Provider != "ollama" || svc.Model != "test-model" {
		t.Errorf("service = %+v", svc)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("config mode = %o", info.Mode().Perm())
	}
}

func TestRunDoctor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`)
	}))
	defer srv.Close()

	path := filepath.Join(t.TempDir(), "config.yml")
	data := fmt.Sprintf("services:\n  test:\n    provider: openai\n    base_url: %s\n    model: model\n", srv.URL)
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := runDoctor([]string{"-config", path}, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ok config", "ok service test", "ok provider configuration", "ok provider connectivity"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q: %s", want, out.String())
		}
	}
}

func TestRunDoctorUsesEnvironmentService(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	data := "service: first\nservices:\n  first:\n    provider: ollama\n    model: one\n  second:\n    provider: ollama\n    model: two\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POUNDAI_SERVICE", "second")
	var out bytes.Buffer
	if err := runDoctor([]string{"-config", path, "-offline"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "ok service second (ollama/two)") {
		t.Errorf("output = %q", out.String())
	}
}

func TestSubcommandHelp(t *testing.T) {
	for _, run := range []func([]string, io.Writer) error{
		func(args []string, out io.Writer) error {
			return runInit(args, strings.NewReader(""), out)
		},
		runDoctor,
	} {
		if err := run([]string{"-h"}, io.Discard); err != nil {
			t.Errorf("help error = %v", err)
		}
	}
}
