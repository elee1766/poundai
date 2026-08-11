package shellctx

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/elee1766/poundai/pkg/config"
)

func TestFormatOSInfo(t *testing.T) {
	got := formatOSInfo("ubuntu", "24.04", "6.8.0")
	want := runtime.GOOS + "/" + runtime.GOARCH + ", ubuntu 24.04, kernel 6.8.0"
	if got != want {
		t.Errorf("formatOSInfo() = %q, want %q", got, want)
	}

	if got := formatOSInfo("", "", ""); got != runtime.GOOS+"/"+runtime.GOARCH {
		t.Errorf("formatOSInfo() fallback = %q", got)
	}
}

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
		{Name: "big", Command: "printf 'xxxxxxxxxxxxxxxxxxxx'", MaxBytes: 10},
	})
	if len(sections) != 1 {
		t.Fatalf("got %d sections", len(sections))
	}
	if len(sections[0].Body) != 10 {
		t.Errorf("body length = %d, want 10", len(sections[0].Body))
	}
}

func TestCommandUsesActiveShell(t *testing.T) {
	t.Setenv("POUNDAI_SHELL", "bash")
	sections := runCommands(context.Background(), []config.ContextCommand{
		{Name: "shell", Command: `printf '%s' "${BASH_VERSION:+bash}"`},
	})
	if len(sections) != 1 || sections[0].Body != "bash" {
		t.Errorf("sections = %+v", sections)
	}
}

func TestCommandShellFallsBack(t *testing.T) {
	t.Setenv("POUNDAI_SHELL", "poundai-missing-shell")
	t.Setenv("SHELL", "bash")
	if got := filepath.Base(commandShell()); got != "bash" {
		t.Errorf("commandShell() = %q", got)
	}
}

func TestCommandInheritsWorkingDirectoryAndEnvironment(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("POUNDAI_TEST_CONTEXT", "inherited")

	sections := runCommands(context.Background(), []config.ContextCommand{
		{Name: "session", Command: `printf '%s\n%s' "$PWD" "$POUNDAI_TEST_CONTEXT"`},
	})
	if len(sections) != 1 || sections[0].Body != dir+"\ninherited" {
		t.Errorf("sections = %+v", sections)
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
