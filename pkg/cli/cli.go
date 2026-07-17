// Package cli implements the zsh_poundai command-line interface.
//
// It reads the ZLE buffer on stdin, a cursor offset as a positional argument,
// asks the configured LLM to complete the command, and prints the result to
// stdout.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/elee1766/zsh_poundai/pkg/complete"
	"github.com/elee1766/zsh_poundai/pkg/config"
	"github.com/elee1766/zsh_poundai/pkg/prompt"
	"github.com/elee1766/zsh_poundai/pkg/provider"
	"github.com/elee1766/zsh_poundai/pkg/shellctx"
)

// Version is set at build time via -ldflags "-X ...pkg/cli.Version=...".
var Version = "dev"

// ExpandPath expands environment variables and a leading ~ in a file path.
func ExpandPath(path string) string {
	path = os.ExpandEnv(path)
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return path
}

// Run is the top-level entry point. It parses flags, reads stdin, loads
// config, calls the LLM, and prints the cleaned completion to stdout.
func Run() error {
	var (
		configPath  = flag.String("config", "", "path to config file (default: $XDG_CONFIG_HOME/zsh_poundai.yaml)")
		service     = flag.String("service", "", "service profile to use (overrides config selector; also settable via $POUNDAI_SERVICE)")
		noContext   = flag.Bool("no-context", false, "skip context gathering")
		debug       = flag.Bool("debug", false, "print the prompt and raw completion to stderr")
		showVersion = flag.Bool("version", false, "print version and exit")
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: zsh_poundai [flags] <cursor>\n\nreads the ZLE buffer on stdin; prints the completion on stdout\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("zsh_poundai", Version)
		return nil
	}

	if flag.NArg() != 1 {
		flag.Usage()
		return fmt.Errorf("expected exactly one argument (cursor offset), got %d", flag.NArg())
	}
	cursor, err := strconv.Atoi(flag.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid cursor offset %q: %w", flag.Arg(0), err)
	}
	if cursor < 0 {
		return fmt.Errorf("cursor offset must be non-negative, got %d", cursor)
	}

	buffer, err := io.ReadAll(io.LimitReader(os.Stdin, 1<<20)) // 1 MiB safety limit
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}
	in := complete.Input{Buffer: string(buffer), Cursor: cursor}

	path := *configPath
	if path == "" {
		if env := os.Getenv("POUNDAI_CONFIG"); env != "" {
			path = env
		} else {
			path = config.DefaultPath()
		}
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}

	svcName := *service
	if svcName == "" {
		svcName = os.Getenv("POUNDAI_SERVICE")
	}
	svc, err := cfg.Active(svcName)
	if err != nil {
		return err
	}
	prov, err := provider.New(svc)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	contextBlock := ""
	if !*noContext {
		contextBlock = shellctx.Render(shellctx.Gather(ctx, cfg.Context))
	}

	// Prompt source: optimized_file artifact (system + few-shot demos) if
	// configured, else prompt.system, else the built-in minimal default.
	var demos []prompt.Demo
	systemOverride := cfg.Prompt.System
	if cfg.Prompt.OptimizedFile != "" {
		opt, err := prompt.Load(ExpandPath(cfg.Prompt.OptimizedFile))
		if err != nil {
			return err
		}
		demos = opt.Demos
		if systemOverride == "" {
			systemOverride = opt.System
		}
	}
	if systemOverride == "" {
		systemOverride = complete.DefaultSystemPrompt
	}
	system := complete.SystemMessage(systemOverride, cfg.Prompt.SystemExtra, contextBlock)
	messages := complete.Messages(system, demos, in)

	if *debug {
		for _, m := range messages {
			fmt.Fprintf(os.Stderr, "--- %s ---\n%s\n", m.Role, m.Content)
		}
		fmt.Fprintf(os.Stderr, "---\n")
	}

	raw, err := prov.Complete(ctx, messages)
	if err != nil {
		return err
	}
	if *debug {
		fmt.Fprintf(os.Stderr, "--- raw completion ---\n%s\n---\n", raw)
	}

	fmt.Print(complete.Clean(raw, in))
	return nil
}
