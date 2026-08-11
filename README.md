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

run the guided setup, then verify the selected provider and model. `init` asks
before replacing an existing config:

```sh
poundai init
poundai doctor
```

the wizard also configures recent history and optional `git` or `files` context
commands. it includes 10 history entries and the `git` preset by default.

use `poundai doctor -offline` to validate configuration and required credential
variables without testing connectivity or credential validity.

releases use utc calver tags: `YYYY.MM.DD`, then `.1`, `.2`, and so on if there
is more than one release that day. every tested push to `master` makes a release,
and you can also run the release workflow manually.

#### arch linux

install poundai from the aur with your preferred helper:

```sh
yay -S poundai
mkdir -p "${XDG_CONFIG_HOME:-$HOME/.config}/poundai"
cp /usr/share/doc/poundai/config.example.yaml "${XDG_CONFIG_HOME:-$HOME/.config}/poundai/config.yml"
```

if you would rather build it yourself:

```sh
make install
mkdir -p "${XDG_CONFIG_HOME:-$HOME/.config}/poundai"
cp config.example.yaml "${XDG_CONFIG_HOME:-$HOME/.config}/poundai/config.yml"
```

then edit `config.yml` and tell poundai which provider and model to use.

#### antidote

to install the binary without modifying your zsh config, run:

```sh
curl -fsSL https://github.com/elee1766/poundai/releases/latest/download/install.sh | POUNDAI_NO_MODIFY_RC=1 sh
```

then add poundai to `${ZDOTDIR:-$HOME}/.zsh_plugins.txt`:

```text
elee1766/poundai
```

antidote will source `poundai.plugin.zsh` the next time it loads your plugins.

### model selection

we support many models, but for a speed/performance balance, i find that gpt-oss-120b and 20b from groq perform very well.

### custom context

poundai can run commands from your current directory on every completion and
add their output to the llm context:

```yaml
context:
  commands:
    - name: git
      command: git status -sb 2>/dev/null
      timeout: 500ms
      max_bytes: 2048
```

commands run concurrently through the active shell, inherit the shell
environment, and can use `pwd` or `$PWD`. poundai ignores failures and empty output.
`-no-context` skips custom commands and all built-in context.

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
