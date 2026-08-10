#!/bin/sh
set -eu

repo=${POUNDAI_REPO:-elee1766/poundai}
install_dir=${POUNDAI_INSTALL_DIR:-"$HOME/.local/bin"}
base_url=${POUNDAI_BASE_URL:-"https://github.com/$repo/releases/download"}

fail() {
    printf 'poundai: %s\n' "$*" >&2
    exit 1
}

case $(uname -s) in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) fail "unsupported operating system: $(uname -s)" ;;
esac

case $(uname -m) in
    x86_64 | amd64) arch=amd64 ;;
    arm64 | aarch64) arch=arm64 ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
esac

if [ -n "${POUNDAI_VERSION:-}" ]; then
    latest=$POUNDAI_VERSION
else
    latest_url=$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest") ||
        fail "could not find the latest release"
    latest=${latest_url##*/}
fi
[ -n "$latest" ] || fail "latest release has no version"

binary="$install_dir/poundai"
asset="poundai-$os-$arch"
config_file=${POUNDAI_CONFIG:-"${XDG_CONFIG_HOME:-$HOME/.config}/poundai/config.yml"}
current=not-installed
if [ -x "$binary" ]; then
    current_output=$($binary -version 2>/dev/null || true)
    case $current_output in
        "poundai "*) current=${current_output#poundai } ;;
    esac
fi

printf 'poundai: installed %s, latest %s\n' "$current" "$latest"

if [ "$current" != "$latest" ] || { [ ! -e "$config_file" ] && [ "${POUNDAI_NO_INSTALL_CONFIG:-0}" != 1 ]; }; then
    tmp=$(mktemp -d "${TMPDIR:-/tmp}/poundai.XXXXXX") || fail "could not create temporary directory"
    trap 'rm -rf "$tmp"' EXIT HUP INT TERM

    curl -fsSL "$base_url/$latest/checksums.txt" -o "$tmp/checksums.txt" || fail "could not download checksums"

    download_verified() {
        name=$1
        curl -fsSL "$base_url/$latest/$name" -o "$tmp/$name" || fail "could not download $name"

        expected=$(
            while read -r checksum checksum_name; do
                checksum_name=${checksum_name#\*}
                checksum_name=${checksum_name#./}
                if [ "$checksum_name" = "$name" ]; then
                    printf '%s\n' "$checksum"
                    break
                fi
            done < "$tmp/checksums.txt"
        )
        [ -n "$expected" ] || fail "release checksum for $name is missing"

        if command -v sha256sum >/dev/null 2>&1; then
            actual=$(sha256sum "$tmp/$name" | cut -d ' ' -f 1)
        elif command -v shasum >/dev/null 2>&1; then
            actual=$(shasum -a 256 "$tmp/$name" | cut -d ' ' -f 1)
        else
            fail "sha256sum or shasum is required"
        fi
        [ "$actual" = "$expected" ] || fail "checksum verification failed for $name"
    }
fi

if [ "$current" != "$latest" ]; then
    download_verified "$asset"
    mkdir -p "$install_dir"
    install -m 755 "$tmp/$asset" "$binary"
    printf 'poundai: installed %s to %s\n' "$latest" "$binary"
else
    printf 'poundai: already up to date\n'
fi

if [ ! -e "$config_file" ] && [ "${POUNDAI_NO_INSTALL_CONFIG:-0}" != 1 ]; then
    download_verified config.example.yaml
    case $config_file in
        */*) mkdir -p "${config_file%/*}" ;;
    esac
    install -m 600 "$tmp/config.example.yaml" "$config_file"
    printf 'poundai: created %s; edit it with your provider and model\n' "$config_file"
fi

shell_path=${SHELL:-}
shell_name=${POUNDAI_SHELL:-${shell_path##*/}}
case $shell_name in
    zsh)
        rc_file=${ZDOTDIR:-$HOME}/.zshrc
        ;;
    bash)
        rc_file=$HOME/.bashrc
        ;;
    *)
        printf 'poundai: add "source <(poundai plugin zsh)" to your shell config\n'
        exit 0
        ;;
esac

if [ -f "$rc_file" ] && {
    grep -F "poundai plugin $shell_name" "$rc_file" >/dev/null 2>&1 ||
        grep -F "poundai.plugin.$shell_name" "$rc_file" >/dev/null 2>&1
}; then
    printf 'poundai: plugin is already sourced from %s\n' "$rc_file"
    exit 0
fi

source_line="source <(poundai plugin $shell_name)"
path_line=
case ":$PATH:" in
    *":$install_dir:"*) ;;
    *) path_line="export PATH=\"$install_dir:\$PATH\"" ;;
esac

if [ "${POUNDAI_NO_MODIFY_RC:-0}" = 1 ]; then
    answer=no
elif [ "${POUNDAI_YES:-0}" = 1 ]; then
    answer=yes
elif [ -t 0 ]; then
    printf 'poundai: add the %s plugin to %s? [y/N] ' "$shell_name" "$rc_file"
    read -r answer || answer=no
elif [ -t 1 ] && [ -r /dev/tty ]; then
    printf 'poundai: add the %s plugin to %s? [y/N] ' "$shell_name" "$rc_file" > /dev/tty
    read -r answer < /dev/tty || answer=no
else
    answer=no
fi

case $answer in
    y | Y | yes | YES)
        mkdir -p "${rc_file%/*}"
        {
            printf '\n# poundai\n'
            [ -z "$path_line" ] || printf '%s\n' "$path_line"
            printf '%s\n' "$source_line"
        } >> "$rc_file"
        printf 'poundai: added plugin setup to %s\n' "$rc_file"
        ;;
    *)
        printf 'poundai: add this to %s when ready:\n' "$rc_file"
        [ -z "$path_line" ] || printf '  %s\n' "$path_line"
        printf '  %s\n' "$source_line"
        ;;
esac
