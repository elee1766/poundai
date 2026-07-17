// Package shellctx gathers extra shell context that gets injected into the
// completion prompt: cwd, OS info, recent history, env vars, and the output
// of user-configured commands.
package shellctx

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/elee1766/zsh_poundai/pkg/config"
)

const (
	defaultCommandTimeout  = 2 * time.Second
	defaultCommandMaxBytes = 4096
	maxHistoryBytes        = 64 * 1024
)

// Section is a single named block of context.
type Section struct {
	Name string
	Body string
}

// Gather collects all configured context sections. Custom commands run
// concurrently; failures are silently skipped (a broken context command
// should never break completion).
func Gather(ctx context.Context, cfg config.Context) []Section {
	var sections []Section

	if cfg.Cwd {
		if wd, err := os.Getwd(); err == nil {
			sections = append(sections, Section{Name: "cwd", Body: wd})
		}
	}
	if cfg.OS {
		sections = append(sections, Section{Name: "os", Body: osInfo()})
	}
	if len(cfg.Env) > 0 {
		var lines []string
		for _, name := range cfg.Env {
			if val, ok := os.LookupEnv(name); ok {
				lines = append(lines, fmt.Sprintf("%s=%s", name, val))
			}
		}
		if len(lines) > 0 {
			sections = append(sections, Section{Name: "environment", Body: strings.Join(lines, "\n")})
		}
	}
	if cfg.History > 0 {
		if hist := recentHistory(cfg.History); hist != "" {
			sections = append(sections, Section{Name: "recent history", Body: hist})
		}
	}

	sections = append(sections, runCommands(ctx, cfg.Commands)...)
	return sections
}

// Render formats sections into a prompt-ready block. Returns "" when there
// are no sections.
func Render(sections []Section) string {
	if len(sections) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("Additional context about the user's shell session:\n")
	for _, s := range sections {
		body := strings.TrimRight(s.Body, "\n")
		if body == "" {
			continue
		}
		fmt.Fprintf(&sb, "<%s>\n%s\n</%s>\n", s.Name, body, s.Name)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func osInfo() string {
	info := runtime.GOOS + "/" + runtime.GOARCH
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if name, ok := strings.CutPrefix(line, "PRETTY_NAME="); ok {
				info += ", " + strings.Trim(name, `"`)
				break
			}
		}
	}
	return info
}

// recentHistory returns the last n commands from the zsh history file.
// The plugin exports POUNDAI_HISTFILE; we fall back to $HISTFILE and then
// ~/.zsh_history. Extended-history timestamps are stripped.
func recentHistory(n int) string {
	path := os.Getenv("POUNDAI_HISTFILE")
	if path == "" {
		path = os.Getenv("HISTFILE")
	}
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		path = home + "/.zsh_history"
	}
	data, err := readTail(path, maxHistoryBytes)
	if err != nil {
		return ""
	}
	lines := parseZshHistory(string(data))
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

func readTail(path string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if st.Size() > maxBytes {
		if _, err := f.Seek(-maxBytes, io.SeekEnd); err != nil {
			return nil, err
		}
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	// If we seeked into the middle of the file, discard bytes up to the first
	// newline to avoid starting on a partial (possibly mid-UTF-8) line.
	if st.Size() > maxBytes {
		if i := strings.IndexByte(string(data), '\n'); i >= 0 && i < len(data)-1 {
			data = data[i+1:]
		}
	}
	return data, nil
}

// parseZshHistory splits history content into commands, handling
// EXTENDED_HISTORY format (": <ts>:<dur>;cmd") and multi-line entries
// (backslash continuation encoded as trailing "\\").
func parseZshHistory(content string) []string {
	var cmds []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		// Continuation of previous multi-line entry.
		if len(cmds) > 0 && strings.HasSuffix(cmds[len(cmds)-1], "\\") {
			cmds[len(cmds)-1] = strings.TrimSuffix(cmds[len(cmds)-1], "\\") + "\n" + line
			continue
		}
		if strings.HasPrefix(line, ": ") {
			if _, cmd, ok := strings.Cut(line, ";"); ok {
				line = cmd
			}
		}
		if line == "" {
			continue
		}
		cmds = append(cmds, line)
	}
	// Drop any unfinished trailing continuation marker.
	if len(cmds) > 0 {
		cmds[len(cmds)-1] = strings.TrimSuffix(cmds[len(cmds)-1], "\\")
	}
	return cmds
}

// runCommands executes user-configured context commands concurrently,
// preserving config order in the result.
func runCommands(ctx context.Context, cmds []config.ContextCommand) []Section {
	results := make([]Section, len(cmds))
	var wg sync.WaitGroup
	for i, cc := range cmds {
		if cc.Command == "" {
			continue
		}
		wg.Add(1)
		go func(i int, cc config.ContextCommand) {
			defer wg.Done()
			results[i] = runCommand(ctx, cc)
		}(i, cc)
	}
	wg.Wait()
	var out []Section
	for _, s := range results {
		if s.Body != "" {
			out = append(out, s)
		}
	}
	return out
}

func runCommand(ctx context.Context, cc config.ContextCommand) Section {
	timeout := cc.Timeout.Std(defaultCommandTimeout)
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, "zsh", "-c", cc.Command)
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return Section{}
	}
	maxBytes := cc.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultCommandMaxBytes
	}
	if len(out) > maxBytes {
		out = out[:maxBytes]
	}
	name := cc.Name
	if name == "" {
		name = cc.Command
	}
	return Section{Name: name, Body: string(out)}
}
