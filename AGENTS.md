# AGENTS.md

## What this is

A zsh plugin (Go binary) that completes shell commands via LLM. The user types a partial command or `# comment`, hits a keybinding, and gets a completion spliced into the zsh line buffer.

## Project layout

```
cmd/poundai/         # thin main: just calls pkg/cli.Run()
poundai.plugin.zsh   # embedded zsh widget that pipes $BUFFER/$CURSOR into the Go binary
poundai.plugin.bash  # embedded bash Readline binding using $READLINE_LINE/$READLINE_POINT
pkg/
  cli/                   # CLI entrypoint logic: flag parsing, config loading, orchestration
  config/                # YAML config loader ($XDG_CONFIG_HOME/poundai/config.yml)
  complete/              # prompt building + output cleaning (Clean strips fences, shebangs, echoed prefixes)
  provider/              # LLM provider implementations (openai, anthropic, gemini, ollama, bedrock)
  shellctx/              # gathers shell context (cwd, OS, history, env, user commands)
  prompt/                # loads prompt artifacts (JSON with system + demos)
```

## Build and test

```sh
make build        # build binary to ./poundai
make test         # go test ./...
make install      # install to ~/.local/bin (override with PREFIX=)
make uninstall    # remove from ~/.local/bin
```

The end-to-end test in `cmd/poundai/main_test.go` builds the binary and runs it against a mock HTTP server. It is skipped with `-short`.

## Go version

`go.mod` requires **Go 1.26.0** (tip/dev). Ensure your Go toolchain matches.

## Config

Config lives at `$XDG_CONFIG_HOME/poundai/config.yml`. The loader uses `yaml.KnownFields(true)` — unknown keys cause hard errors. See `config.example.yaml` for the full schema.

`context.commands` run concurrently through the active shell; their stdout is appended to shell context for every completion.

API keys resolve via `api_key` (literal) or `api_key_env` (env var name). The active service is selected by `service:` in config, `$POUNDAI_SERVICE` env, or `-service` flag.

## Key conventions

- The binary reads the shell editing buffer from **stdin** and cursor offset in **bytes** as the **first positional arg**.
- Output is printed to stdout and spliced into the shell editing buffer at the cursor position.
- `complete.Clean()` is where output post-processing happens (fence stripping, shebang removal, prefix deduplication, comment-to-newline). This is a frequent source of edge cases — check `complete_test.go` for the 15+ test scenarios.
- Provider constructors validate all required fields (model, API key) eagerly — fail fast, not at completion time.
- Cloud OpenAI-compatible providers (openai, groq, mistral, openrouter) require an API key; local endpoints (custom `base_url`) do not.
- `ExtraHeaders` cannot overwrite auth headers (`Authorization`, `x-api-key`).
- Gemini API key errors are redacted to avoid leaking credentials.
- Bedrock initializes its AWS client once at construction, not per-call.
- Provider implementations share a `postJSON` helper and `splitSystem` for APIs that take the system prompt out-of-band (Anthropic, Gemini).
- Shell context commands run concurrently with per-command timeouts and byte limits.
- Set version at build time: `go build -ldflags "-X github.com/elee1766/poundai/pkg/cli.Version=v1.0.0" ./cmd/poundai`.
