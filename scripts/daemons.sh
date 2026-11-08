#!/bin/sh
# Builds the Linux hustlexd the app embeds, to install on servers.
set -eu
cd "$(dirname "$0")/.."
go=${GO:-go}
for arch in amd64 arm64; do
	GOOS=linux GOARCH=$arch CGO_ENABLED=0 $go build -trimpath -ldflags "-s -w" -o hostconn/daemons/hustlexd-linux-$arch ./cmd/hustlexd
	gzip -9f hostconn/daemons/hustlexd-linux-$arch
done
