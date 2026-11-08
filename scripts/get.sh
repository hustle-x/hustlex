#!/bin/sh
# Installs the latest hustleX on Linux for the current user (no root):
#   curl -fsSL https://github.com/hustle-x/hustlex/releases/latest/download/get.sh | sh
set -eu
repo='hustle-x/hustlex'
version='1.0.0'
case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) echo "get.sh: unsupported architecture $(uname -m)" >&2; exit 1 ;;
esac
[ "$(uname -s)" = Linux ] || { echo "get.sh: this installer is for Linux; see https://github.com/$repo/releases/latest" >&2; exit 1; }
base="https://github.com/$repo/releases/latest/download"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
fetch() { if command -v curl >/dev/null 2>&1; then curl -fsSL "$1" -o "$2"; else wget -qO "$2" "$1"; fi; }
echo "Downloading hustleX $version for linux/$arch…"
fetch "$base/hustlex-$version-linux-$arch.tar.gz" "$work/hustlex.tar.gz"
fetch "$base/install.sh" "$work/install.sh"
sh "$work/install.sh" "$work/hustlex.tar.gz"
