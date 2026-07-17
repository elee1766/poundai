# zsh_poundai

AI command completion for zsh, in a single Go binary. Type a partial command
or a `# comment describing what you want`, hit your keybinding, and the
completion is spliced into your buffer at the cursor.

A Go reimplementation of [zsh_codex](https://github.com/tom-doerr/zsh_codex)
with:

- **No Python required** — one static binary, instant startup
- **Local providers** — native Ollama support, plus anything
  OpenAI-compatible (llama.cpp, vLLM, LM Studio, ...)
- **Cloud providers** — OpenAI, Groq, Mistral, OpenRouter, Gemini, Anthropic,
  Amazon Bedrock
- **Custom injected context** — cwd, OS, recent shell history, env vars, and
  the output of arbitrary commands (e.g. `git status -sb`), all opt-in and
  sandboxed by timeouts
- **YAML config** with multiple named profiles

## Install

```sh
go install github.com/elee1766/zsh_poundai@latest
```

Then source the plugin and bind a key in your `.zshrc`:

```sh
source /path/to/zsh_poundai/zsh_poundai.plugin.zsh
bindkey '^X' create_poundai_completion   # Ctrl-X to complete
```

Or with oh-my-zsh, clone into `~/.oh-my-zsh/custom/plugins/zsh_poundai` and
add `zsh_poundai` to your `plugins=(...)`.

## Configure

Copy [`config.example.yaml`](config.example.yaml) to
`~/.config/zsh_poundai.yaml` and pick a service. Minimal local setup:

```yaml
services:
  local:
    provider: ollama
    model: qwen2.5-coder:7b
```

Minimal OpenAI setup:

```yaml
services:
  openai:
    provider: openai
    api_key_env: OPENAI_API_KEY
    model: gpt-4o-mini
```

### Providers

| `provider` | Notes |
|---|---|
| `ollama` | Native API, defaults to `http://localhost:11434`, no key |
| `openai` | Also covers llama.cpp/vLLM/LM Studio/any compatible API via `base_url` |
| `groq`, `mistral`, `openrouter` | OpenAI-compatible with preset endpoints |
| `gemini` | Google Generative Language API |
| `anthropic` | Anthropic Messages API |
| `bedrock` | Amazon Bedrock Converse API, standard AWS credential chain, set `region` |

Common service keys: `model`, `base_url`, `api_key` / `api_key_env`,
`temperature`, `max_tokens`, `timeout`, `extra_headers`.

### Context injection

Everything under `context:` is injected into the system prompt:

```yaml
context:
  cwd: true          # current directory
  os: true           # OS/arch + distro
  history: 10        # last N commands from $HISTFILE
  env: [VIRTUAL_ENV] # selected env vars
  commands:          # arbitrary command output
    - name: git
      command: git status -sb 2>/dev/null
      timeout: 500ms
      max_bytes: 2048
```

Context commands run concurrently with per-command timeouts; failures are
silently skipped so a broken command never blocks completion. Unlike
zsh_codex's `ZSH_CODEX_PREEXECUTE_COMMENT`, nothing typed into your buffer is
ever executed.

### Prompt customization

```yaml
prompt:
  system_extra: "Prefer long flags. This machine runs NixOS."
  # system: "..."   # fully replace the built-in system prompt
```

## Usage

- `ls` + keybinding → completes flags/arguments at the cursor
- `# find files modified in the last day` + keybinding → command inserted on
  the next line
- Works mid-line: text after the cursor is preserved and de-duplicated

Switch profiles per shell with `export ZSH_POUNDAI_SERVICE=groq`, or point at
an alternate config with `ZSH_POUNDAI_CONFIG=...`.

Debug what gets sent:

```sh
echo -n "# list docker containers" | zsh_poundai -debug 25
```

## CLI

```
usage: zsh_poundai [flags] <cursor>

reads the ZLE buffer on stdin; prints the completion on stdout

  -config string     path to config file (default: $XDG_CONFIG_HOME/zsh_poundai.yaml)
  -service string    service profile to use (overrides config selector)
  -no-context        skip context gathering
  -debug             print the prompt and raw completion to stderr
```

## Prompt tuning

The built-in system prompt is minimal on purpose: in our evals, strong models
(gpt-oss-20b and up) scored best with it (~90%), and rule-heavy prompts made
them worse. Small local models (<= ~14B) benefit from a richer prompt with
few-shot examples; supply one as a JSON artifact:

```yaml
prompt:
  optimized_file: /path/to/prompt.example.json   # +5-15 pts on small models
```

The artifact format is `{"system": "...", "demos": [{"comment": "...", "command": "..."}]}`.

## Development

```sh
go build ./...
go test ./...
```

## License

MIT
