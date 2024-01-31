#!/bin/sh

set -e

redwood_install="${HOME}/.redwood"
bin_dir="$redwood_install/bin"
exe="$bin_dir/rw"

if [ ! -d "$bin_dir" ]; then
	mkdir -p "$bin_dir"
fi

# TODO(jgmw): Do these still need to be lowercased in this case?
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m | tr '[:upper:]' '[:lower:]')

# TODO(jgmw): Update to the correct url when that is clear
curl --fail --location --progress-bar --output "$exe.zip" "https://example.com/$os/$arch/latest.zip"

unzip -d "$bin_dir" -o "$exe.zip"
chmod +x "$exe"

rm "$exe.zip"

echo "Redwood CLI (rw) was installed successfully to $exe"

if command -v rw >/dev/null; then
    echo "Run 'rw --help' to get started"
else
    case $SHELL in
    /bin/zsh) shell_profile=".zshrc" ;;
    *) shell_profile=".bash_profile" ;;
    esac
    echo "Manually add the directory to your \$HOME/$shell_profile (or similar)"
    echo "  export REDWOOD_INSTALL=\"$redwood_install\""
    echo "  export PATH=\"\$REDWOOD_INSTALL/bin:\$PATH\""
    echo "Run '$exe --help' to get started"
fi