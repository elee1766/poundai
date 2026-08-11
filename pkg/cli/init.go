package cli

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	poundai "github.com/elee1766/poundai"
	"github.com/elee1766/poundai/pkg/config"
	"gopkg.in/yaml.v3"
)

type initOptions struct {
	configPath     string
	name           string
	provider       string
	baseURL        string
	apiKeyEnv      string
	model          string
	region         string
	history        int
	commands       string
	force          bool
	nonInteractive bool
}

var providerDefaults = map[string]struct {
	baseURL   string
	model     string
	apiKeyEnv string
}{
	"ollama":     {baseURL: "http://localhost:11434", model: "qwen2.5-coder:7b"},
	"openai":     {model: "gpt-4o-mini", apiKeyEnv: "OPENAI_API_KEY"},
	"groq":       {model: "llama-3.3-70b-versatile", apiKeyEnv: "GROQ_API_KEY"},
	"mistral":    {model: "codestral-latest", apiKeyEnv: "MISTRAL_API_KEY"},
	"openrouter": {apiKeyEnv: "OPENROUTER_API_KEY"},
	"gemini":     {model: "gemini-2.0-flash", apiKeyEnv: "GEMINI_API_KEY"},
	"anthropic":  {model: "claude-3-5-haiku-latest", apiKeyEnv: "ANTHROPIC_API_KEY"},
	"bedrock":    {},
}

func runInit(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stdout)
	var opts initOptions
	fs.StringVar(&opts.configPath, "config", "", "path to config file")
	fs.StringVar(&opts.name, "name", "", "service profile name")
	fs.StringVar(&opts.provider, "provider", "", "provider name")
	fs.StringVar(&opts.baseURL, "base-url", "", "provider base URL")
	fs.StringVar(&opts.apiKeyEnv, "api-key-env", "", "environment variable containing the API key")
	fs.StringVar(&opts.model, "model", "", "model name")
	fs.StringVar(&opts.region, "region", "", "AWS region for Bedrock")
	fs.IntVar(&opts.history, "history", 10, "number of recent history entries to include (0 disables history)")
	fs.StringVar(&opts.commands, "commands", "git", "context command presets to include: git, files, or none")
	fs.BoolVar(&opts.force, "force", false, "replace an existing config without confirmation")
	fs.BoolVar(&opts.nonInteractive, "non-interactive", false, "use flags and defaults without prompting")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("init: unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	historySet := false
	commandsSet := false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "history":
			historySet = true
		case "commands":
			commandsSet = true
		}
	})

	reader := bufio.NewReader(stdin)
	path := initConfigPath(opts.configPath)
	existing, readErr := os.ReadFile(path)
	untouchedExample := readErr == nil && bytes.Equal(existing, []byte(poundai.ConfigExample()))
	if readErr == nil && !opts.force && !untouchedExample {
		if opts.nonInteractive {
			return fmt.Errorf("config already exists: %s (use -force to replace it)", path)
		}
		replace, err := ask(reader, stdout, "Config already exists; replace it?", "no")
		if err != nil {
			return err
		}
		replace = strings.ToLower(replace)
		if replace != "y" && replace != "yes" {
			return fmt.Errorf("config not changed")
		}
	} else if readErr != nil && !os.IsNotExist(readErr) {
		return readErr
	}

	if opts.name == "" {
		opts.name = "default"
	}
	if opts.provider == "" {
		if opts.nonInteractive {
			opts.provider = "ollama"
		} else {
			var err error
			opts.provider, err = ask(reader, stdout, "Provider (ollama, openai, groq, mistral, openrouter, gemini, anthropic, bedrock)", "ollama")
			if err != nil {
				return err
			}
		}
	}
	opts.provider = strings.ToLower(opts.provider)
	defaults, ok := providerDefaults[opts.provider]
	if !ok {
		return fmt.Errorf("unknown provider %q", opts.provider)
	}
	if opts.baseURL == "" && promptsForBaseURL(opts.provider) {
		if opts.nonInteractive {
			opts.baseURL = defaults.baseURL
		} else {
			var err error
			opts.baseURL, err = ask(reader, stdout, "Base URL", defaults.baseURL)
			if err != nil {
				return err
			}
		}
	}
	if opts.model == "" {
		if opts.nonInteractive {
			opts.model = defaults.model
		} else {
			var err error
			opts.model, err = ask(reader, stdout, "Model", defaults.model)
			if err != nil {
				return err
			}
		}
	}
	if opts.model == "" {
		return fmt.Errorf("model is required")
	}
	if opts.provider == "bedrock" {
		if opts.region == "" {
			opts.region = "us-east-1"
		}
	} else if opts.apiKeyEnv == "" && needsAPIKey(opts.provider, opts.baseURL) {
		if opts.nonInteractive {
			opts.apiKeyEnv = defaults.apiKeyEnv
		} else {
			var err error
			opts.apiKeyEnv, err = ask(reader, stdout, "API key environment variable", defaults.apiKeyEnv)
			if err != nil {
				return err
			}
		}
	}
	if !opts.nonInteractive && !historySet {
		value, err := ask(reader, stdout, "Recent history entries to include (0 disables history)", "10")
		if err != nil {
			return err
		}
		opts.history, err = strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid history count %q", value)
		}
	}
	if opts.history < 0 {
		return fmt.Errorf("history count must be non-negative")
	}
	if !opts.nonInteractive && !commandsSet {
		value, err := ask(reader, stdout, "Context commands to include (git, files, or none; comma-separated)", "git")
		if err != nil {
			return err
		}
		opts.commands = value
	}
	contextCommands, err := commandPresets(opts.commands)
	if err != nil {
		return err
	}

	svc := config.Service{
		Provider:  opts.provider,
		BaseURL:   opts.baseURL,
		APIKeyEnv: opts.apiKeyEnv,
		Model:     opts.model,
		Region:    opts.region,
	}
	cfg := config.Config{
		Service:  opts.name,
		Services: map[string]config.Service{opts.name: svc},
		Context: config.Context{
			Cwd:      true,
			OS:       true,
			History:  opts.history,
			Commands: contextCommands,
		},
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	if err := writeConfig(path, data); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	if _, err := fmt.Fprintf(stdout, "Created %s\nRun 'poundai doctor' to verify the setup.\n", path); err != nil {
		return err
	}
	return nil
}

func initConfigPath(path string) string {
	if path == "" {
		path = os.Getenv("POUNDAI_CONFIG")
	}
	if path == "" {
		return config.UserPath()
	}
	return ExpandPath(path)
}

func writeConfig(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".config.yml.*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func ask(reader *bufio.Reader, out io.Writer, label, defaultValue string) (string, error) {
	if defaultValue == "" {
		fmt.Fprintf(out, "%s: ", label)
	} else {
		fmt.Fprintf(out, "%s [%s]: ", label, defaultValue)
	}
	value, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		value = defaultValue
	}
	return value, nil
}

func needsAPIKey(providerName, baseURL string) bool {
	if providerName == "ollama" || providerName == "bedrock" {
		return false
	}
	return providerName != "openai" || baseURL == ""
}

func promptsForBaseURL(providerName string) bool {
	return providerName == "ollama" || providerName == "openai"
}

func commandPresets(value string) ([]config.ContextCommand, error) {
	presets := map[string]config.ContextCommand{
		"git": {
			Name:     "git",
			Command:  "git status -sb 2>/dev/null",
			Timeout:  config.Duration(500 * time.Millisecond),
			MaxBytes: 2048,
		},
		"files": {
			Name:     "files",
			Command:  "find . -type f | head -50",
			Timeout:  config.Duration(500 * time.Millisecond),
			MaxBytes: 4096,
		},
	}
	names := strings.Split(strings.ToLower(value), ",")
	commands := make([]config.ContextCommand, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "none" && len(names) == 1 {
			return []config.ContextCommand{}, nil
		}
		command, ok := presets[name]
		if !ok {
			return nil, fmt.Errorf("unknown context command preset %q (want git, files, or none)", name)
		}
		commands = append(commands, command)
	}
	return commands, nil
}
