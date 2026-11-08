#!/usr/bin/env bash
# Builds release artifacts locally (no CI minutes): a macOS universal app,
# ad-hoc signed, and hustlexd for Linux servers. Usage: scripts/release-local.sh 0.0.1-alpha.1
# Needs: Go 1.27+, zip, and rcodesign (cargo install apple-codesign) when not on macOS.
set -euo pipefail

version="${1:?usage: $0 <version>}"
root="$(cd "$(dirname "$0")/.." && pwd)"
dist="$root/dist"
mkdir -p "$dist"

cd "$root"
make daemons
go tool mygo build -platform darwin/universal cmd/hustlex

app="$root/build/darwin-universal/hustleX.app"
if [[ "$(uname)" == "Darwin" ]]; then
  codesign --force --deep --sign - "$app"
else
  rcodesign sign "$app"
fi
(cd "$root/build/darwin-universal" && zip -qry "$dist/hustleX-$version-macos-universal.zip" hustleX.app)

for arch in amd64 arm64; do
  mkdir -p "$dist/linux-$arch"
  GOOS=linux GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$dist/linux-$arch/hustlexd" ./cmd/hustlexd
  tar -C "$dist/linux-$arch" -czf "$dist/hustlexd-$version-linux-$arch.tar.gz" hustlexd
done

(cd "$dist" && sha256sum hustleX-"$version"-*.zip hustlexd-"$version"-*.tar.gz > SHA256SUMS)
echo "Artifacts in $dist"
