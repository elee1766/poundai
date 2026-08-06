# zsh_poundai

AI command generation for Zsh and Bash. A fast provider and a capable coding
model such as `gpt-oss-20b` are recommended.

## Install

```sh
make install
cp config.example.yaml ~/.config/zsh_poundai.yaml
```

Edit the copied configuration with your provider and model settings.

## Zsh

Source the plugin from `~/.zshrc`:

```zsh
source /path/to/zsh_poundai/zsh_poundai.plugin.zsh
```

Type `# <prompt>` and press Enter. The comment is saved to history and the
generated command appears at the next prompt for review. For mid-line
completion, bind `create_poundai_completion` to a key:

```zsh
bindkey '^X' create_poundai_completion
```

## Bash

Source the plugin from `~/.bashrc`:

```bash
source /path/to/zsh_poundai/bash_poundai.plugin.bash
```

Type a partial command or `# <prompt>`, then press Ctrl-X Ctrl-A. The generated
text is inserted at the cursor. For a comment prompt it is placed on the next
line; review it and press Enter to execute it.

Bash Readline does not provide a safe way for a shell function to conditionally
delegate to its original Enter action, so the Bash plugin uses a separate key
binding rather than overriding Enter.

To choose another binding, add this after sourcing the plugin:

```bash
bind -x '"\C-g":create_poundai_completion'
```

Shell-specific overrides are available through `ZSH_POUNDAI_BIN`,
`ZSH_POUNDAI_SERVICE`, and `ZSH_POUNDAI_CONFIG`, or their `BASH_POUNDAI_*`
equivalents.
