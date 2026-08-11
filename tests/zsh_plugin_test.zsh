#!/usr/bin/env zsh
set -e

repo_root=${0:A:h:h}
source "$repo_root/poundai.plugin.zsh"

fake_poundai() {
    local input
    input=$(<&0)
    [[ $input == $expected_input ]]
    [[ $1 == $expected_cursor ]]
    [[ $POUNDAI_SHELL == zsh ]]
    print -n ' world'
}

ZSH_POUNDAI_BIN=fake_poundai
expected_input='echo héllo'
expected_cursor=11
_poundai_generate 'echo héllo' 10
[[ $REPLY == ' world' ]]

expected_input='echo héllo tail'
expected_cursor=11
_poundai_generate 'echo héllo tail' 10
[[ $REPLY == ' world' ]]
