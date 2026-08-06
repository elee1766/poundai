# bash_poundai: AI command completion for Bash.
#
# Source this file from ~/.bashrc, then type a partial command or
# "# describe what you want" and press Ctrl-X Ctrl-A. The generated text is
# inserted at the cursor for review before execution.

BASH_POUNDAI_BIN="${BASH_POUNDAI_BIN:-zsh_poundai}"

_poundai_bash_generate() {
    local text=$1 cursor=$2 errfile rc
    local service=${BASH_POUNDAI_SERVICE:-${POUNDAI_SERVICE:-}}
    local config=${BASH_POUNDAI_CONFIG:-${POUNDAI_CONFIG:-}}

    errfile=$(mktemp "${TMPDIR:-/tmp}/bash_poundai.XXXXXX") || return 1
    POUNDAI_REPLY=$(printf '%s' "$text" | \
        POUNDAI_HISTFILE="${HISTFILE:-}" \
        POUNDAI_SERVICE="$service" \
        POUNDAI_CONFIG="$config" \
        POUNDAI_SHELL=bash \
        "$BASH_POUNDAI_BIN" "$cursor" 2>"$errfile")
    rc=$?

    if (( rc != 0 )); then
        printf 'bash_poundai: %s\n' "$(tail -n 1 "$errfile" 2>/dev/null)" >&2
        rm -f "$errfile"
        return 1
    fi
    rm -f "$errfile"
}

_poundai_bash_insert() {
    local insertion=$1
    local LC_ALL=C
    local prefix=${READLINE_LINE:0:READLINE_POINT}
    local suffix=${READLINE_LINE:READLINE_POINT}

    READLINE_LINE="${prefix}${insertion}${suffix}"
    (( READLINE_POINT += ${#insertion} ))
}

create_poundai_completion() {
    printf '\nbash_poundai: thinking...\n' >&2
    if ! _poundai_bash_generate "$READLINE_LINE" "$READLINE_POINT"; then
        return 1
    fi
    [[ -z $POUNDAI_REPLY ]] && return 0
    _poundai_bash_insert "$POUNDAI_REPLY"
}

if [[ $- == *i* ]]; then
    bind -x '"\C-x\C-a":create_poundai_completion'
fi
