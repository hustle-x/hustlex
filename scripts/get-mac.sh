#!/bin/sh
# Installs the latest hustleX on macOS (Apple silicon):
#   curl -fsSL https://github.com/hustle-x/hustlex/releases/latest/download/get-mac.sh | sh
set -eu
repo='hustle-x/hustlex'
dest="${INSTALL_DIR:-/Applications}"
[ "$(uname -s)" = Darwin ] || { echo "get-mac.sh: this installer is for macOS" >&2; exit 1; }
[ "$(uname -m)" = arm64 ] || { echo "get-mac.sh: only Apple silicon builds exist for now" >&2; exit 1; }
work="$(mktemp -d)"
mnt="$work/mnt"
cleanup() { hdiutil detach "$mnt" -quiet >/dev/null 2>&1 || true; rm -rf "$work"; }
trap cleanup EXIT
echo "Downloading hustleX…"
curl -fL --progress-bar "https://github.com/$repo/releases/latest/download/hustleX-1.0.0-arm64.dmg" -o "$work/hustlex.dmg"
mkdir -p "$mnt"
hdiutil attach -nobrowse -readonly -mountpoint "$mnt" "$work/hustlex.dmg" >/dev/null
[ -d "$mnt/hustleX.app" ] || { echo "get-mac.sh: hustleX.app is not in the disk image" >&2; exit 1; }
mkdir -p "$dest"
rm -rf "$dest/hustleX.app"
cp -R "$mnt/hustleX.app" "$dest/"
# The build is not notarized yet: drop the quarantine mark so the first launch is not blocked.
xattr -dr com.apple.quarantine "$dest/hustleX.app" 2>/dev/null || true
echo "Installed to $dest/hustleX.app"
