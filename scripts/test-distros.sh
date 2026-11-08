#!/bin/sh
# Runs hustleX's daemon and its setup's system packages step in a container
# of each Linux distribution it supports, with docker. Usage:
#   scripts/test-distros.sh [image ...]
set -eu
cd "$(dirname "$0")/.."
images=${*:-"ubuntu:24.04 debian:12 fedora:41 archlinux:latest alpine:3.21 opensuse/tumbleweed"}
arch=$(docker info --format '{{.Architecture}}')
case "$arch" in
x86_64 | amd64) goarch=amd64 ;;
aarch64 | arm64) goarch=arm64 ;;
*) echo "unknown docker architecture $arch" >&2; exit 1 ;;
esac
# Under the home folder: docker VMs (colima, Docker Desktop) share it.
mkdir -p "$HOME/.cache"
out=$(mktemp -d "$HOME/.cache/hustlex-distros.XXXXXX")
trap 'rm -rf "$out"' EXIT
CGO_ENABLED=0 GOOS=linux GOARCH=$goarch go build -o "$out/hustlexd" ./cmd/hustlexd
CGO_ENABLED=0 GOOS=linux GOARCH=$goarch go test -c -o "$out/setup.test" ./core/setup
cp hostconn/prepare.sh "$out/"
# Images there are no native build of run emulated.
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$out/hustlexd-amd64" ./cmd/hustlexd
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -c -o "$out/setup-amd64.test" ./core/setup
failed=""
for image in $images; do
	platform="" suffix=""
	if [ "$image" = archlinux:latest ] && [ "$goarch" = arm64 ]; then
		platform="--platform=linux/amd64" suffix="-amd64" # Arch has no arm64 image
	fi
	echo "== $image $platform"
	# The daemon serves and answers its command line; then the setup step
	# installs what it needs, as root, and a second plan has nothing left.
	if ! docker run --rm $platform -v "$out:/hustlex:ro" -e HUSTLEX_DISTRO_TEST=1 -e SUFFIX="$suffix" "$image" sh -c '
		set -e
		if [ -n "$SUFFIX" ] && [ -f /etc/pacman.conf ]; then
			# Emulated, pacman cannot sandbox its downloads (seccomp).
			sed -i "s/^\[options\]/[options]\nDisableSandbox/" /etc/pacman.conf
		fi
		sh /hustlex/prepare.sh
		cp /hustlex/hustlexd$SUFFIX /tmp/hustlexd && cp /hustlex/setup$SUFFIX.test /tmp/setup.test
		export HUSTLEX_HOME=/tmp/enx HOME=/root SHELL=/bin/sh
		/tmp/hustlexd serve >/tmp/serve.log 2>&1 &
		for i in $(seq 1 150); do /tmp/hustlexd ls >/dev/null 2>&1 && break; sleep 0.2; done
		mkdir -p /tmp/p
		id=$(/tmp/hustlexd spawn --agent shell --project /tmp/p "echo terminal-works")
		for i in $(seq 1 150); do /tmp/hustlexd read "$id" | grep -q "^terminal-works" && break; sleep 0.2; done
		/tmp/hustlexd read "$id" | grep -q "^terminal-works"
		/tmp/hustlexd ls
		/tmp/hustlexd stop
		rc=0; /tmp/setup.test -test.run TestSystemPackagesOnThisDistro -test.v >/tmp/test.log 2>&1 || rc=$?
		tail -5 /tmp/test.log
		exit $rc
	'; then
		failed="$failed $image"
	fi
done
if [ -n "$failed" ]; then
	echo "failed:$failed" >&2
	exit 1
fi
echo "all passed"
