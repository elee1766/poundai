#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
source "$repo_root/bash_poundai.plugin.bash"

fake_poundai() {
    local input
    IFS= read -r input || [[ -n $input ]]
    [[ $input == abc ]]
    [[ $1 == 3 ]]
    [[ $POUNDAI_SHELL == bash ]]
    [[ $POUNDAI_HISTFILE == /tmp/bash-poundai-history ]]
    [[ $POUNDAI_SERVICE == test-service ]]
    [[ $POUNDAI_CONFIG == /tmp/bash-poundai-config ]]
    printf ' completion'
}

BASH_POUNDAI_BIN=fake_poundai
BASH_POUNDAI_SERVICE=test-service
BASH_POUNDAI_CONFIG=/tmp/bash-poundai-config
HISTFILE=/tmp/bash-poundai-history
_poundai_bash_generate abc 3
[[ $POUNDAI_REPLY == ' completion' ]]

READLINE_LINE='echo héllo'
READLINE_POINT=11
_poundai_bash_insert ' world'
[[ $READLINE_LINE == 'echo héllo world' ]]
[[ $READLINE_POINT == 17 ]]

READLINE_LINE='# list files'
READLINE_POINT=12
_poundai_bash_insert $'\nls -la'
[[ $READLINE_LINE == $'# list files\nls -la' ]]
[[ $READLINE_POINT == 19 ]]
