package shellctx

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elee1766/poundai/pkg/config"
)

func TestParseZshHistory(t *testing.T) {
	content := ": 1700000000:0;ls -la\n" +
		": 1700000001:2;git status\n" +
		"plain command\n" +
		": 1700000002:0;echo one \\\\\n" +
		"two\n"
	got := parseZshHistory(content)
	want := []string{"ls -la", "git status", "plain command", "echo one \\\ntwo"}
	if len(got) != len(want) {
		t.Fatalf("got %d entries %q, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRecentHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hist")
	var sb strings.Builder
	for i := 0; i < 20; i++ {
		sb.WriteString(": 1700000000:0;command-")
		sb.WriteByte(byte('a' + i))
		sb.WriteString("\n")
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POUNDAI_HISTFILE", path)

	got := recentHistory(3)
	want := "command-r\ncommand-s\ncommand-t"
	if got != want {
		t.Errorf("recentHistory(3) = %q, want %q", got, want)
	}
}

func TestGatherAndRender(t *testing.T) {
	t.Setenv("POUNDAI_TEST_VAR", "hello")
	cfg := config.Context{
		Cwd: true,
		OS:  true,
		Env: []string{"POUNDAI_TEST_VAR", "POUNDAI_MISSING_VAR"},
		Commands: []config.ContextCommand{
			{Name: "greeting", Command: "echo hi there"},
			{Name: "broken", Command: "exit 1"},
			{Name: "empty", Command: "true"},
		},
	}
	sections := Gather(context.Background(), cfg)
	rendered := Render(sections)

	wd, _ := os.Getwd()
	for _, want := range []string{
		"<cwd>\n" + wd,
		"<os>",
		"POUNDAI_TEST_VAR=hello",
		"<greeting>\nhi there\n</greeting>",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered context missing %q:\n%s", want, rendered)
		}
	}
	for _, reject := range []string{"POUNDAI_MISSING_VAR", "<broken>", "<empty>"} {
		if strings.Contains(rendered, reject) {
			t.Errorf("rendered context should not contain %q:\n%s", reject, rendered)
		}
	}
}

func TestRenderEmpty(t *testing.T) {
	if got := Render(nil); got != "" {
		t.Errorf("Render(nil) = %q", got)
	}
}

func TestCommandMaxBytes(t *testing.T) {
	sections := runCommands(context.Background(), []config.ContextCommand{
		{Name: "big", Command: "printf 'x%.0s' {1..100}", MaxBytes: 10},
	})
	if len(sections) != 1 {
		t.Fatalf("got %d sections", len(sections))
	}
	if len(sections[0].Body) != 10 {
		t.Errorf("body length = %d, want 10", len(sections[0].Body))
	}
}

func TestCommandTimeout(t *testing.T) {
	sections := runCommands(context.Background(), []config.ContextCommand{
		{Name: "slow", Command: "sleep 5; echo done", Timeout: config.Duration(50 * 1e6)}, // 50ms
	})
	if len(sections) != 0 {
		t.Errorf("timed-out command should produce no section, got %+v", sections)
	}
}

func TestContextHook(t *testing.T) {
	hookPath := filepath.Join(t.TempDir(), "context")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\nprintf 'project=%s' \"$POUNDAI_HOOK_TEST\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POUNDAI_HOOK_TEST", "custom")

	sections := Gather(context.Background(), config.Context{Hook: hookPath})
	if len(sections) != 1 {
		t.Fatalf("got %d sections, want 1: %+v", len(sections), sections)
	}
	if sections[0].Name != "custom context" || sections[0].Body != "project=custom" {
		t.Errorf("hook section = %+v", sections[0])
	}
}

func TestContextHookFailureIsSkipped(t *testing.T) {
	hookPath := filepath.Join(t.TempDir(), "context")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}

	if sections := Gather(context.Background(), config.Context{Hook: hookPath}); len(sections) != 0 {
		t.Errorf("failed hook should produce no section, got %+v", sections)
	}
}
