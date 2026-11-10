#!/bin/sh
# Installs the latest hustleX on Linux for the current user (no root):
#   curl -fsSL https://hustle-x.github.io/hustlex/get.sh | sh
set -eu
repo='hustle-x/hustlex'
version='1.0.2'
base="https://github.com/$repo/releases/latest/download"

if [ -t 1 ]; then G=$(printf '\033[32m'); D=$(printf '\033[2m'); B=$(printf '\033[1m'); W=$(printf '\033[97m'); N=$(printf '\033[0m'); else G=''; D=''; B=''; W=''; N=''; fi
banner() {
	echo
	awk -v w="$W" -v d="$D" -v n="$N" 'BEGIN{srand()} {s=""; for(i=1;i<=length($0);i++){c=substr($0,i,1); if(c=="#") s=s w int(rand()*2) n; else s=s c} print "  " s}' <<'MASK'
 ###                                  #   ###             ###     ###
 ###                                ###   ###              ###   ####
 ###                                ###   ###              #### ####
 ### ####    ###    ##    ######   ###### ###    #####      #######
 #########   ###    ###  ########  ###### ###   #######      #####
 ####   ###  ###    ### ###    ##   ###   ###  ###   ###      ####
 ###    ###  ###    ###  ######     ###   ###  ###   ###     #####
 ###    ###  ###    ###  ########   ###   ###  #########    #######
 ###    ###  ###   ####     ######  ###   ###  ###          ### ####
 ###    ###  ###   #### ###    ###  ###   ###  ###   ###   ###   ###
 ###    ###  ##########  ########   ##### ###   ########  ####   ####
 ###    ###   ##### ##    ######     #### ###    ######  ####     ####
MASK
	echo
	printf '  %severy agent that codes, in one place.%s\n' "$B" "$N"
	printf '  %s%s%s\n\n' "$D" "$1" "$N"
}
step() { printf '  %s✓%s %-24s %s%s%s\n' "$G" "$N" "$1" "$D" "$2" "$N"; }
now() { date +%s; }

case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) echo "get.sh: unsupported architecture $(uname -m)" >&2; exit 1 ;;
esac
[ "$(uname -s)" = Linux ] || { echo "get.sh: this installer is for Linux; see https://github.com/$repo/releases/latest" >&2; exit 1; }
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
fetch() { if command -v curl >/dev/null 2>&1; then curl -fsSL "$1" -o "$2"; else wget -qO "$2" "$1"; fi; }
sha() { if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{print $1}'; else shasum -a 256 "$1" | awk '{print $1}'; fi; }
t0=$(now)
file="hustlex-$version-linux-$arch.tar.gz"
banner "v$version · Linux · $arch"
step "Checking system" "$(. /etc/os-release 2>/dev/null && echo "${PRETTY_NAME:-Linux}" || echo Linux) · $(uname -m)"
fetch "$base/$file" "$work/$file"
step "Downloading hustleX" "$(du -h "$work/$file" | awk '{print $1}')"
fetch "$base/SHA256SUMS" "$work/SHA256SUMS"
want=$(awk -v f="$file" '$2==f{print $1}' "$work/SHA256SUMS")
[ -n "$want" ] && [ "$want" = "$(sha "$work/$file")" ] || { echo "get.sh: the download does not match SHA256SUMS" >&2; exit 1; }
step "Verifying archive" "sha256 ok"
fetch "$base/install.sh" "$work/install.sh"
sh "$work/install.sh" "$work/$file" >"$work/install.log" 2>&1 || { cat "$work/install.log" >&2; exit 1; }
step "Installing" "~/.local/hustlex.app"
printf '\n  Done in %ss. Open hustleX from your app menu.\n  To remove it later: sh install.sh --uninstall\n\n  Docs   https://hustle-x.github.io/hustlex/\n\n' "$(( $(now) - t0 ))"
