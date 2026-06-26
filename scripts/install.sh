#!/bin/sh
# Daedalus installer for Linux and macOS.
#
# Quick install (latest release):
#   curl -fsSL https://raw.githubusercontent.com/Codigo-de-Altura/Daedalus/main/scripts/install.sh | sh
#
# Options (environment variables):
#   DAEDALUS_VERSION       install a specific tag (e.g. v0.1.0); default: latest
#   DAEDALUS_INSTALL_DIR   install directory; default: /usr/local/bin if writable,
#                          otherwise $HOME/.local/bin
#
# What it does: downloads the matching archive from GitHub Releases, verifies its
# SHA-256 checksum, extracts the `daedalus` binary, and places it on your PATH.
set -eu

REPO="Codigo-de-Altura/Daedalus"
BINARY="daedalus"

info() { printf '\033[1;34m==>\033[0m %s\n' "$1"; }
warn() { printf '\033[1;33mwarning:\033[0m %s\n' "$1" >&2; }
fail() { printf '\033[1;31merror:\033[0m %s\n' "$1" >&2; exit 1; }

# --- detect platform ---
os=$(uname -s)
case "$os" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) fail "unsupported OS: $os (on Windows use scripts/install.ps1)" ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) fail "unsupported architecture: $arch" ;;
esac

# --- pick a downloader ---
if command -v curl >/dev/null 2>&1; then
  dl() { curl -fsSL "$1" -o "$2"; }
  fetch() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
  dl() { wget -qO "$2" "$1"; }
  fetch() { wget -qO- "$1"; }
else
  fail "need curl or wget to download"
fi

# --- pick a checksum tool ---
if command -v sha256sum >/dev/null 2>&1; then
  sha() { sha256sum "$1" | awk '{print $1}'; }
elif command -v shasum >/dev/null 2>&1; then
  sha() { shasum -a 256 "$1" | awk '{print $1}'; }
else
  sha() { echo "skip"; }
fi

# --- resolve the release tag ---
tag="${DAEDALUS_VERSION:-}"
if [ -z "$tag" ]; then
  info "Resolving latest release..."
  tag=$(fetch "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | head -n1 \
    | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/')
  [ -n "$tag" ] || fail "could not find a published release yet — build from source instead (see the docs)"
fi
version="${tag#v}"

asset="${BINARY}_${version}_${os}_${arch}.tar.gz"
checksums="${BINARY}_${version}_checksums.txt"
base="https://github.com/${REPO}/releases/download/${tag}"

# --- choose an install directory ---
bindir="${DAEDALUS_INSTALL_DIR:-}"
if [ -z "$bindir" ]; then
  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
    bindir=/usr/local/bin
  else
    bindir="$HOME/.local/bin"
  fi
fi
mkdir -p "$bindir" || fail "cannot create install directory: $bindir"

# --- download, verify, extract, install ---
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

info "Downloading ${asset} (${tag})..."
dl "${base}/${asset}" "${tmp}/${asset}" || fail "download failed: ${base}/${asset}"

want=""
if dl "${base}/${checksums}" "${tmp}/${checksums}" 2>/dev/null; then
  want=$(grep " ${asset}\$" "${tmp}/${checksums}" | awk '{print $1}')
fi
got=$(sha "${tmp}/${asset}")
if [ "$got" = "skip" ]; then
  warn "no sha256 tool available; skipping checksum verification"
elif [ -n "$want" ]; then
  [ "$want" = "$got" ] || fail "checksum mismatch for ${asset}"
  info "Checksum verified."
else
  warn "checksum file not found; skipping verification"
fi

info "Extracting..."
tar -xzf "${tmp}/${asset}" -C "$tmp" || fail "failed to extract ${asset}"
binpath=$(find "$tmp" -type f -name "$BINARY" | head -n1)
[ -n "$binpath" ] || fail "binary '${BINARY}' not found in archive"

chmod 0755 "$binpath"
mv "$binpath" "${bindir}/${BINARY}" || fail "failed to install to ${bindir} (try DAEDALUS_INSTALL_DIR=\$HOME/.local/bin)"

info "Installed ${BINARY} ${tag} -> ${bindir}/${BINARY}"

# --- PATH hint ---
case ":$PATH:" in
  *":$bindir:"*) ;;
  *)
    warn "${bindir} is not on your PATH. Add it to your shell profile:"
    printf '    export PATH="%s:$PATH"\n' "$bindir" >&2
    ;;
esac

"${bindir}/${BINARY}" --version 2>/dev/null || true
