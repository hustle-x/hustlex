#!/bin/sh
# Installs the latest hustleX on macOS (Apple silicon):
#   curl -fsSL https://hustle-x.github.io/hustlex/get-mac.sh | sh
set -eu
repo='hustle-x/hustlex'
version='1.3.3'
file="hustleX-$version-arm64.dmg"
base="https://github.com/$repo/releases/latest/download"
dest="${INSTALL_DIR:-/Applications}"

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

[ "$(uname -s)" = Darwin ] || { echo "get-mac.sh: this installer is for macOS" >&2; exit 1; }
[ "$(uname -m)" = arm64 ] || { echo "get-mac.sh: only Apple silicon builds exist for now" >&2; exit 1; }
work="$(mktemp -d)"
mnt="$work/mnt"
trap 'hdiutil detach "$mnt" -quiet >/dev/null 2>&1 || true; rm -rf "$work"' EXIT
t0=$(now)
banner "v$version · macOS · Apple silicon"
step "Checking system" "macOS $(sw_vers -productVersion) · $(uname -m)"
printf '  %sDownloading hustleX…%s\n' "$D" "$N"
curl -fL# "$base/$file" -o "$work/$file"
[ -t 1 ] && printf '\033[2A\033[J'
size=$(du -h "$work/$file" | awk '{print $1}')
step "Downloading hustleX" "$size"
want=$(curl -fsSL "$base/SHA256SUMS" | awk -v f="$file" '$2==f{print $1}')
got=$(shasum -a 256 "$work/$file" | awk '{print $1}')
[ -n "$want" ] && [ "$want" = "$got" ] || { echo "get-mac.sh: the download does not match SHA256SUMS" >&2; exit 1; }
step "Verifying disk image" "sha256 ok"
mkdir -p "$mnt"
hdiutil attach -nobrowse -readonly -mountpoint "$mnt" "$work/$file" >/dev/null
[ -d "$mnt/hustleX.app" ] || { echo "get-mac.sh: hustleX.app is not in the disk image" >&2; exit 1; }
mkdir -p "$dest"
rm -rf "$dest/hustleX.app"
cp -R "$mnt/hustleX.app" "$dest/"
step "Installing" "$dest/hustleX.app"
# Not notarized yet: drop the quarantine mark so the first launch is not blocked.
xattr -dr com.apple.quarantine "$dest/hustleX.app" 2>/dev/null || true
step "Clearing quarantine" "first launch will not be blocked"
printf '\n  Done in %ss. Open hustleX from Launchpad, or run:\n    %sopen -a hustleX%s\n\n  Docs   https://hustle-x.github.io/hustlex/\n\n' "$(( $(now) - t0 ))" "$B" "$N"
