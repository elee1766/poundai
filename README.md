# poundai

poundai is a way to get llm-generated commands for any terminal, with your existing shell

<video src="assets/demo-1.webm" controls title="poundai command completion demo"></video>

## setup guide

### Install

Download the latest release archive for your platform. For example, on Linux
amd64:

```sh
curl -fLO https://github.com/elee1766/poundai/releases/latest/download/poundai-linux-amd64.tar.gz
tar -xzf poundai-linux-amd64.tar.gz
install -Dm755 poundai-linux-amd64/poundai "$HOME/.local/bin/poundai"
install -Dm644 poundai-linux-amd64/poundai.plugin.zsh "${XDG_DATA_HOME:-$HOME/.local/share}/poundai/poundai.plugin.zsh"
install -Dm644 poundai-linux-amd64/poundai.plugin.bash "${XDG_DATA_HOME:-$HOME/.local/share}/poundai/poundai.plugin.bash"
install -Dm644 poundai-linux-amd64/config.example.yaml "${XDG_CONFIG_HOME:-$HOME/.config}/poundai/config.yml"
```

Release archives include the Zsh and Bash plugins and `config.example.yaml`.
Available platform names are `linux-amd64`, `linux-arm64`, `darwin-amd64`, and
`darwin-arm64`; substitute one of them in the URL and commands above.
Releases use UTC CalVer tags (`YYYY.MM.DD`, with a numeric suffix when more than
one release is made on the same day). Every push to `master` publishes a release
after tests pass; the Release workflow can also be run manually. To build from
source instead:

```sh
make install
mkdir -p "${XDG_CONFIG_HOME:-$HOME/.config}/poundai"
cp config.example.yaml "${XDG_CONFIG_HOME:-$HOME/.config}/poundai/config.yml"
```

Edit the copied configuration with your provider and model settings.

### Custom Context

For context that cannot be expressed through `context.commands`, configure an
executable hook. Poundai runs it for each completion from the current working
directory and appends its stdout to the model context:

```sh
mkdir -p "${XDG_CONFIG_HOME:-$HOME/.config}/poundai"
cat > "${XDG_CONFIG_HOME:-$HOME/.config}/poundai/context" <<'EOF'
#!/bin/sh
printf 'active project: %s\n' "${PROJECT_NAME:-unknown}"
EOF
chmod +x "${XDG_CONFIG_HOME:-$HOME/.config}/poundai/context"
```

Then add its path to `config.yml`:

```yaml
context:
  hook: context
```

Relative hook paths are resolved from the directory containing `config.yml`;
environment variables and a leading `~` are also expanded. Failures and empty
output are ignored; execution is limited to two seconds and 4 KiB of stdout.
`-no-context` disables the hook along with built-in context.

### Zsh

Source the plugin from `~/.zshrc`:

```zsh
source "${XDG_DATA_HOME:-$HOME/.local/share}/poundai/poundai.plugin.zsh"
```

Type `# <prompt>` and press Enter. The comment is saved to history and the
generated command appears at the next prompt for review. For mid-line
completion, bind `create_poundai_completion` to a key:

this is the recommended way to use poundai.

```zsh
bindkey '^X' create_poundai_completion
```

### Bash

a bash plugin exists, but i strongly recommend using zsh. there is no nice way to override the enter key in bash.

source the plugin from `~/.bashrc`:

```bash
source "${XDG_DATA_HOME:-$HOME/.local/share}/poundai/poundai.plugin.bash"
```

Type a partial command or `# <prompt>`, then press Ctrl-X Ctrl-A. The generated
text is inserted at the cursor. For a comment prompt it is placed on the next
line; review it and press Enter to execute it.

To choose another binding, add this after sourcing the plugin:

```bash
bind -x '"\C-g":create_poundai_completion'
```

Shell-specific overrides are available through `ZSH_POUNDAI_BIN`,
`ZSH_POUNDAI_SERVICE`, and `ZSH_POUNDAI_CONFIG`, or their `BASH_POUNDAI_*`
equivalents.
