# poundai

poundai is a way to get llm-generated commands for any terminal, with your existing shell

so the idea is instead of waiting for long response from harness/pulling up web ui/copy pasting output, you can recall commands from a llm, without leaving the terminal you want to run the command.

![poundai command completion demo](assets/demo-1.gif)

## setup

### install

the installer finds the latest release, compares it with what you already have,
checks the download, and installs or updates the binary. it also detects zsh or
bash and asks before touching your shell config:

```sh
curl -fsSL https://github.com/elee1766/poundai/releases/latest/download/install.sh | sh
```

set `POUNDAI_INSTALL_DIR` to install somewhere other than `~/.local/bin`, or
`POUNDAI_NO_MODIFY_RC=1` to leave shell config alone. releases contain one
standalone binary for linux or macos on amd64 or arm64. both plugins are bundled
into that binary. on the first install, the script also puts the example config
at `${XDG_CONFIG_HOME:-$HOME/.config}/poundai/config.yml` for you to edit.

releases use utc calver tags: `YYYY.MM.DD`, then `.1`, `.2`, and so on if there
is more than one release that day. every tested push to `master` makes a release,
and you can also run the release workflow manually.

if you would rather build it yourself:

```sh
make install
mkdir -p "${XDG_CONFIG_HOME:-$HOME/.config}/poundai"
cp config.example.yaml "${XDG_CONFIG_HOME:-$HOME/.config}/poundai/config.yml"
```

then edit `config.yml` and tell poundai which provider and model to use.

### model selection

we support many models, but for a speed/performance balance, i find that gpt-oss-120b and 20b from groq perform very well.

### custom context

poundai can run an executable hook when `context.commands` is not enough. it
runs from your current directory on every completion, and whatever it prints is
added to the llm context:

```sh
mkdir -p "${XDG_CONFIG_HOME:-$HOME/.config}/poundai"
cat > "${XDG_CONFIG_HOME:-$HOME/.config}/poundai/context" <<'EOF'
#!/bin/sh
printf 'active project: %s\n' "${PROJECT_NAME:-unknown}"
EOF
chmod +x "${XDG_CONFIG_HOME:-$HOME/.config}/poundai/context"
```

then point at it from `config.yml`:

```yaml
context:
  hook: context
```

relative paths start from the directory containing `config.yml`. environment
variables and a leading `~` work too. poundai ignores failures and empty output,
and cuts the hook off after two seconds or 4 KiB of output. `-no-context` skips
the hook and all built-in context.

### zsh

add this to `~/.zshrc`:

```zsh
source <(poundai plugin zsh)
```

type `# <prompt>` and press enter. poundai keeps the comment in your history and
puts the generated command at the next prompt so you can check it before running
it.

you can also bind `create_poundai_completion` to a key and use it anywhere in a
command. this is the recommended way to use poundai:

```zsh
bindkey '^X' create_poundai_completion
```

### bash

a bash plugin exists, but i strongly recommend zsh. bash does not give us a nice
way to take over the enter key.

add this to `~/.bashrc`:

```bash
source <(poundai plugin bash)
```

type a partial command or `# <prompt>`, then press ctrl-x ctrl-a. poundai inserts
the result at your cursor. comment prompts put it on the next line so you can
check it before pressing enter.

if you want another binding, add it after sourcing the plugin:

```bash
bind -x '"\C-g":create_poundai_completion'
```

you can override the binary, service, or config for one shell with
`ZSH_POUNDAI_BIN`, `ZSH_POUNDAI_SERVICE`, and `ZSH_POUNDAI_CONFIG`. bash has the
same variables under `BASH_POUNDAI_*`.
