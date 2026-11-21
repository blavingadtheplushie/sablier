#!/bin/sh
# installs sablier: curl -fsSL https://sablier.hackclub.cc/install.sh | sh
#
#   SABLIER_VERSION      a release tag like v0.1.0, default the latest
#   SABLIER_INSTALL_DIR  where to put it, default ~/.local/bin
set -eu

repo="espcaa/sablier"
dir="${SABLIER_INSTALL_DIR:-$HOME/.local/bin}"

fail() {
	echo "sablier: $*" >&2
	exit 1
}

case "$(uname -s)" in
Darwin) os=darwin ;;
Linux) os=linux ;;
*) fail "no build for $(uname -s), on windows download sablier_windows_amd64.exe from https://github.com/$repo/releases" ;;
esac

case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) fail "no build for $(uname -m)" ;;
esac

if [ -n "${SABLIER_VERSION:-}" ]; then
	base="https://github.com/$repo/releases/download/$SABLIER_VERSION"
else
	base="https://github.com/$repo/releases/latest/download"
fi
base="${SABLIER_RELEASE_URL:-$base}"
file="sablier_${os}_${arch}"

fetch() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$1" -o "$2"
	elif command -v wget >/dev/null 2>&1; then
		wget -qO "$2" "$1"
	else
		fail "needs curl or wget"
	fi
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "downloading $file"
fetch "$base/$file" "$tmp/$file" || fail "couldn't download $base/$file"
fetch "$base/checksums.txt" "$tmp/checksums.txt" || fail "couldn't download the checksums"

want="$(grep " $file\$" "$tmp/checksums.txt" | cut -d' ' -f1)"
if command -v sha256sum >/dev/null 2>&1; then
	got="$(sha256sum "$tmp/$file" | cut -d' ' -f1)"
else
	got="$(shasum -a 256 "$tmp/$file" | cut -d' ' -f1)"
fi
[ -n "$want" ] && [ "$want" = "$got" ] || fail "checksum mismatch for $file, not installing it"

mkdir -p "$dir"
chmod +x "$tmp/$file"
mv "$tmp/$file" "$dir/sablier"
echo "installed $("$dir/sablier" version) to $dir/sablier"

case ":$PATH:" in
*":$dir:"*) ;;
*) echo "add $dir to your PATH to run it as sablier" ;;
esac
echo "next: sablier login"
