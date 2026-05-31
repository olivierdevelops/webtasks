#!/bin/sh
# webtasks installer
#
#   curl -fsSL https://olivierdevelops.github.io/webtasks/install.sh | sh
#
# Installs the `webtasks` binary into a directory on your PATH. Downloads a
# prebuilt release asset when available; otherwise builds from source with Go.
#
# Environment overrides:
#   WEBTASKS_INSTALL_DIR   target directory (default: /usr/local/bin or ~/.local/bin)
#   WEBTASKS_VERSION       release tag to install (default: latest)

set -eu

REPO="olivierdevelops/webtasks"
BINARY="webtasks"
VERSION="${WEBTASKS_VERSION:-latest}"

# ── pretty output ───────────────────────────────────────────────────────────
if [ -t 1 ]; then
  BOLD="$(printf '\033[1m')"; DIM="$(printf '\033[2m')"
  GREEN="$(printf '\033[32m')"; RED="$(printf '\033[31m')"
  YELLOW="$(printf '\033[33m')"; RESET="$(printf '\033[0m')"
else
  BOLD=""; DIM=""; GREEN=""; RED=""; YELLOW=""; RESET=""
fi
info() { printf '%s==>%s %s\n' "$GREEN" "$RESET" "$1"; }
warn() { printf '%swarning:%s %s\n' "$YELLOW" "$RESET" "$1" >&2; }
err()  { printf '%serror:%s %s\n' "$RED" "$RESET" "$1" >&2; exit 1; }

need() { command -v "$1" >/dev/null 2>&1; }

# ── detect platform ───────────────────────────────────────────────────────────
detect_platform() {
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$os" in
    linux)  OS="linux" ;;
    darwin) OS="darwin" ;;
    *) err "unsupported OS: $os (build from source: https://github.com/$REPO)" ;;
  esac
  case "$arch" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) err "unsupported architecture: $arch" ;;
  esac
}

# ── pick an install dir on PATH ───────────────────────────────────────────────
pick_install_dir() {
  if [ -n "${WEBTASKS_INSTALL_DIR:-}" ]; then
    INSTALL_DIR="$WEBTASKS_INSTALL_DIR"
  elif [ -w "/usr/local/bin" ] 2>/dev/null; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="$HOME/.local/bin"
  fi
  mkdir -p "$INSTALL_DIR" 2>/dev/null || err "cannot create install dir: $INSTALL_DIR"
}

# ── download helpers ──────────────────────────────────────────────────────────
fetch() {
  # fetch <url> <output-path>
  if need curl; then
    curl -fsSL "$1" -o "$2"
  elif need wget; then
    wget -qO "$2" "$1"
  else
    err "need curl or wget to download"
  fi
}

release_asset_url() {
  # Resolves the download URL for the platform asset.
  name="${BINARY}_${OS}_${ARCH}.tar.gz"
  if [ "$VERSION" = "latest" ]; then
    echo "https://github.com/$REPO/releases/latest/download/$name"
  else
    echo "https://github.com/$REPO/releases/download/$VERSION/$name"
  fi
}

install_from_release() {
  url="$(release_asset_url)"
  tmp="$(mktemp -d)"
  info "Downloading $BINARY ($OS/$ARCH)..."
  if ! fetch "$url" "$tmp/$BINARY.tar.gz" 2>/dev/null; then
    rm -rf "$tmp"
    return 1
  fi
  tar -xzf "$tmp/$BINARY.tar.gz" -C "$tmp" 2>/dev/null || { rm -rf "$tmp"; return 1; }
  if [ ! -f "$tmp/$BINARY" ]; then rm -rf "$tmp"; return 1; fi
  install -m 0755 "$tmp/$BINARY" "$INSTALL_DIR/$BINARY"
  rm -rf "$tmp"
  return 0
}

install_from_source() {
  need git || err "no prebuilt binary available and 'git' is not installed"
  need go  || err "no prebuilt binary available and 'go' is not installed (need Go 1.25+)"
  info "No prebuilt binary found — building from source with Go..."
  tmp="$(mktemp -d)"
  git clone --depth 1 "https://github.com/$REPO.git" "$tmp/src" >/dev/null 2>&1 \
    || err "git clone failed"
  ( cd "$tmp/src" && go build -trimpath -ldflags '-s -w' -o "$tmp/$BINARY" ./cmd/webtasks ) \
    || err "go build failed"
  install -m 0755 "$tmp/$BINARY" "$INSTALL_DIR/$BINARY"
  rm -rf "$tmp"
}

# ── PATH hint ─────────────────────────────────────────────────────────────────
path_hint() {
  case ":$PATH:" in
    *":$INSTALL_DIR:"*) : ;;
    *) warn "$INSTALL_DIR is not on your PATH. Add it:"
       printf '    %sexport PATH="%s:$PATH"%s\n' "$DIM" "$INSTALL_DIR" "$RESET" >&2 ;;
  esac
}

main() {
  printf '%swebtasks installer%s\n\n' "$BOLD" "$RESET"
  detect_platform
  pick_install_dir

  if ! install_from_release; then
    install_from_source
  fi

  info "Installed ${BOLD}$BINARY${RESET} to $INSTALL_DIR/$BINARY"
  path_hint

  printf '\n%sNext steps:%s\n' "$BOLD" "$RESET"
  printf '  %s# grab the demo bundle (38 example tasks)%s\n' "$DIM" "$RESET"
  printf '  git clone --depth 1 https://github.com/%s ~/webtasks\n' "$REPO"
  printf '  %s# start the server, pointed at a bundle%s\n' "$DIM" "$RESET"
  printf '  WEBTASKS_BUNDLE=~/webtasks/demo %s &\n' "$BINARY"
  printf '  %s# call a task%s\n' "$DIM" "$RESET"
  printf '  curl -s -X POST localhost:8765/tasks/basics/title -d "{}"\n'
  printf '\nDocs: https://olivierdevelops.github.io/webtasks/\n'
}

main "$@"
