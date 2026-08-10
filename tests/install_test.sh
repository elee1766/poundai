#!/bin/sh
set -eu

repo_root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
tmp=$(mktemp -d "${TMPDIR:-/tmp}/poundai-install-test.XXXXXX")
trap 'rm -rf "$tmp"' EXIT HUP INT TERM

version=2099.01.02
release_dir="$tmp/releases/$version"
install_dir="$tmp/bin"
case $(uname -s) in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) exit 0 ;;
esac
case $(uname -m) in
    x86_64 | amd64) arch=amd64 ;;
    arm64 | aarch64) arch=arm64 ;;
    *) exit 0 ;;
esac
asset="poundai-$os-$arch"
mkdir -p "$release_dir"

cat > "$release_dir/$asset" <<EOF
#!/bin/sh
if [ "\${1:-}" = -version ]; then
    printf 'poundai %s\n' '$version'
fi
EOF
chmod +x "$release_dir/$asset"
printf 'service: test\n' > "$release_dir/config.example.yaml"

if command -v sha256sum >/dev/null 2>&1; then
    binary_checksum=$(sha256sum "$release_dir/$asset" | cut -d ' ' -f 1)
    config_checksum=$(sha256sum "$release_dir/config.example.yaml" | cut -d ' ' -f 1)
else
    binary_checksum=$(shasum -a 256 "$release_dir/$asset" | cut -d ' ' -f 1)
    config_checksum=$(shasum -a 256 "$release_dir/config.example.yaml" | cut -d ' ' -f 1)
fi
printf '%s  ./%s\n%s  ./config.example.yaml\n' \
    "$binary_checksum" "$asset" "$config_checksum" > "$release_dir/checksums.txt"

HOME="$tmp/home" \
SHELL=/bin/zsh \
POUNDAI_VERSION="$version" \
POUNDAI_BASE_URL="file://$tmp/releases" \
POUNDAI_INSTALL_DIR="$install_dir" \
POUNDAI_NO_MODIFY_RC=1 \
    sh "$repo_root/install.sh"

got=$("$install_dir/poundai" -version)
[ "$got" = "poundai $version" ]
[ -f "$tmp/home/.config/poundai/config.yml" ]
[ ! -e "$tmp/home/.zshrc" ]

output=$(HOME="$tmp/home" \
    SHELL=/bin/zsh \
    POUNDAI_VERSION="$version" \
    POUNDAI_BASE_URL="file://$tmp/releases" \
    POUNDAI_INSTALL_DIR="$install_dir" \
    POUNDAI_NO_MODIFY_RC=1 \
    sh "$repo_root/install.sh")
case $output in
    *"already up to date"*) ;;
    *) printf 'missing up-to-date message:\n%s\n' "$output" >&2; exit 1 ;;
esac

HOME="$tmp/home" \
SHELL=/bin/zsh \
POUNDAI_VERSION="$version" \
POUNDAI_BASE_URL="file://$tmp/releases" \
POUNDAI_INSTALL_DIR="$install_dir" \
POUNDAI_YES=1 \
    sh "$repo_root/install.sh"

grep -F 'source <(poundai plugin zsh)' "$tmp/home/.zshrc" >/dev/null

HOME="$tmp/home" \
SHELL=/bin/zsh \
POUNDAI_VERSION="$version" \
POUNDAI_BASE_URL="file://$tmp/releases" \
POUNDAI_INSTALL_DIR="$install_dir" \
POUNDAI_YES=1 \
    sh "$repo_root/install.sh"

[ "$(grep -Fc 'source <(poundai plugin zsh)' "$tmp/home/.zshrc")" -eq 1 ]

printf 'tampered\n' >> "$release_dir/$asset"
if HOME="$tmp/bad-home" \
    SHELL=/bin/zsh \
    POUNDAI_VERSION="$version" \
    POUNDAI_BASE_URL="file://$tmp/releases" \
    POUNDAI_INSTALL_DIR="$tmp/bad-bin" \
    POUNDAI_NO_INSTALL_CONFIG=1 \
    POUNDAI_NO_MODIFY_RC=1 \
    sh "$repo_root/install.sh" >/dev/null 2>&1; then
    printf 'installer accepted an invalid checksum\n' >&2
    exit 1
fi
[ ! -e "$tmp/bad-bin/poundai" ]
