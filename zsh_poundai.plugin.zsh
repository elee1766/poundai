# zsh_poundai: AI command completion for zsh.
#
# Installation:
#   1. Build/install the binary:  make install  (or go install ./cmd/zsh_poundai)
#   2. Source this file from your .zshrc (or install as an oh-my-zsh plugin)
#
# Usage:
#   Type "# describe what you want" and press Enter. The comment is preserved
#   and the generated command appears below it, ready to review and execute.
#   You can also bind create_poundai_completion to a key for mid-line completion:
#     bindkey '^X' create_poundai_completion
#
# Configuration lives in $XDG_CONFIG_HOME/zsh_poundai.yaml (see config.example.yaml).
#
# Environment overrides:
#   ZSH_POUNDAI_BIN      path to the zsh_poundai binary (default: zsh_poundai on $PATH)
#   ZSH_POUNDAI_SERVICE  service profile to use for this shell
#   ZSH_POUNDAI_CONFIG   alternate config file path

typeset -g ZSH_POUNDAI_BIN="${ZSH_POUNDAI_BIN:-zsh_poundai}"

# --- Syntax highlighting fix -------------------------------------------------
# Enable interactive comments so "# ..." is a comment, not a command.
# We set it immediately AND via a one-shot precmd hook so it takes effect
# even if this plugin loads before zsh-syntax-highlighting or
# fast-syntax-highlighting.
setopt INTERACTIVE_COMMENTS

_poundai_highlight_setup() {
    setopt INTERACTIVE_COMMENTS
    # Style comments as grey for both major highlighting plugins.
    if (( $+ZSH_HIGHLIGHT_STYLES )); then
        ZSH_HIGHLIGHT_STYLES[comment]="${ZSH_HIGHLIGHT_STYLES[comment]:-fg=8}"
    fi
    if (( $+FAST_HIGHLIGHT_STYLES )); then
        FAST_HIGHLIGHT_STYLES[comment]="${FAST_HIGHLIGHT_STYLES[comment]:-fg=8}"
    fi
    add-zsh-hook -d precmd _poundai_highlight_setup  # run once then remove
}
autoload -Uz add-zsh-hook
add-zsh-hook precmd _poundai_highlight_setup

# --- Core --------------------------------------------------------------------

# _poundai_generate runs the binary on the given text/cursor and sets $REPLY.
# This is a plain function (not a ZLE widget) so it can be called from any context.
# Returns 0 on success, 1 on error.
_poundai_generate() {
    local text=$1 cursor=$2 errfile
    local service=${ZSH_POUNDAI_SERVICE:-${POUNDAI_SERVICE:-}}
    local config=${ZSH_POUNDAI_CONFIG:-${POUNDAI_CONFIG:-}}
    errfile=$(mktemp "${TMPDIR:-/tmp}/zsh_poundai.XXXXXX") || return 1

    REPLY=$(printf '%s' "$text" | \
        POUNDAI_HISTFILE="$HISTFILE" \
        POUNDAI_SERVICE="$service" \
        POUNDAI_CONFIG="$config" \
        POUNDAI_SHELL=zsh \
        "$ZSH_POUNDAI_BIN" "$cursor" 2>"$errfile")
    local rc=$?

    if (( rc != 0 )); then
        print -u2 "zsh_poundai: $(tail -n 1 "$errfile" 2>/dev/null)"
        rm -f "$errfile"
        return 1
    fi
    rm -f "$errfile"
    return 0
}

# accept-line override: if the buffer is a single-line comment, accept it
# (so the comment alone goes into history), generate the command, and push
# it onto the input buffer for the next prompt. The result is two separate
# history entries: the comment and the command you chose to run.
_poundai_accept_line() {
    if [[ "$BUFFER" == \#* && "$BUFFER" != *$'\n'* ]]; then
        local comment="$BUFFER"
        local cursor="$CURSOR"

        # Accept the comment line — it goes into history as-is.
        zle .accept-line

        # Show a thinking indicator on its own line.
        print "\nzsh_poundai: thinking..."

        # Generate the command (runs outside ZLE context now).
        if _poundai_generate "$comment" "$cursor"; then
            # Strip leading newline (Clean prepends \n for comment lines).
            REPLY="${REPLY#$'\n'}"
            if [[ -n "$REPLY" ]]; then
                # Push the generated command onto the input buffer.
                # It will appear at the next prompt, ready to edit/execute.
                print -z -- "$REPLY"
            fi
        fi

        # Move up and clear the thinking line — prompt redraws over it.
        print -n "\033[A\r\033[K"
    else
        zle .accept-line
    fi
}

zle -N accept-line _poundai_accept_line

# Manual completion widget: splice at cursor position (for mid-line use).
# Bind with: bindkey '^X' create_poundai_completion
create_poundai_completion() {
    zle -M "zsh_poundai: thinking..."
    zle -R

    if ! _poundai_generate "$BUFFER" "$CURSOR"; then
        zle -M "zsh_poundai: generation failed"
        return 1
    fi
    zle -M ""
    [[ -z "$REPLY" ]] && return 0

    local prefix=${BUFFER:0:$CURSOR}
    local suffix=${BUFFER:$CURSOR}
    BUFFER="${prefix}${REPLY}${suffix}"
    CURSOR=$(( CURSOR + ${#REPLY} ))
    zle redisplay
}

zle -N create_poundai_completion
