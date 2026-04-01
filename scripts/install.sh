#!/usr/bin/env bash
set -euo pipefail

BINARY="gitcompare"
REPO="anthonygacis/git-compare"
GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"
INSTALL_DIR=""

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

info() { echo -e "${CYAN}${BOLD}=>${RESET} $*"; }
ok()   { echo -e "${GREEN}✓${RESET} $*"; }
warn() { echo -e "${YELLOW}!${RESET} $*"; }
die()  { echo -e "${RED}error:${RESET} $*" >&2; exit 1; }

detect_platform() {
    OS="$(uname -s)"
    ARCH="$(uname -m)"

    case "$OS" in
        Darwin) OS="darwin" ;;
        Linux)  OS="linux"  ;;
        *)      die "Unsupported OS: $OS. On Windows use: irm https://raw.githubusercontent.com/${REPO}/main/install.ps1 | iex" ;;
    esac

    case "$ARCH" in
        x86_64)        ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *)             die "Unsupported architecture: $ARCH" ;;
    esac
}

resolve_install_dir() {
    if [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
    elif echo "$PATH" | tr ':' '\n' | grep -qx "$HOME/.local/bin"; then
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
    else
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
        warn "$INSTALL_DIR is not in your PATH."
        warn "Add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        warn "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
}

check_downloader() {
    if command -v curl &>/dev/null; then
        DOWNLOADER="curl"
    elif command -v wget &>/dev/null; then
        DOWNLOADER="wget"
    else
        die "Neither curl nor wget found. Please install one and re-run."
    fi
}

fetch() {
    local url="$1" dest="$2"
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL "$url" -o "$dest"
    else
        wget -qO "$dest" "$url"
    fi
}

fetch_stdout() {
    local url="$1"
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL "$url"
    else
        wget -qO- "$url"
    fi
}

get_latest_version() {
    info "Fetching latest release info..."
    local response
    response="$(fetch_stdout "$GITHUB_API")"

    if command -v python3 &>/dev/null; then
        VERSION="$(echo "$response" | python3 -c "import sys,json; print(json.load(sys.stdin)['tag_name'])")"
    elif command -v jq &>/dev/null; then
        VERSION="$(echo "$response" | jq -r '.tag_name')"
    else
        VERSION="$(echo "$response" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
    fi

    [ -n "$VERSION" ] || die "Could not determine latest release version. Check https://github.com/${REPO}/releases"
    ok "Latest release: $VERSION"
}

download_binary() {
    local asset_name

    if [ "$OS" = "darwin" ]; then
        asset_name="${BINARY}-darwin-universal"
    else
        asset_name="${BINARY}-${OS}-${ARCH}"
    fi

    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${asset_name}"

    info "Downloading ${asset_name} ..."
    TMP_FILE="$(mktemp)"
    trap 'rm -f "$TMP_FILE"' EXIT

    fetch "$DOWNLOAD_URL" "$TMP_FILE" || die "Download failed. Check that the release has an asset named '${asset_name}':\n  https://github.com/${REPO}/releases/tag/${VERSION}"

    chmod +x "$TMP_FILE"
    ok "Downloaded $VERSION"
}

install_binary() {
    info "Installing to ${INSTALL_DIR}/${BINARY} ..."

    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY}"
    else
        sudo mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY}"
    fi

    ok "Installed ${INSTALL_DIR}/${BINARY}"
}

verify() {
    if command -v "$BINARY" &>/dev/null; then
        INSTALLED_PATH="$(command -v "$BINARY")"
        ok "$BINARY is ready at $INSTALLED_PATH"
    else
        warn "$BINARY installed to $INSTALL_DIR but your PATH may need refreshing."
        warn "Run:  source ~/.zshrc  (or ~/.bashrc), then try again."
    fi
}

echo -e "\n${BOLD}${CYAN}gitcompare installer${RESET}\n"

detect_platform
info "Platform: ${OS}/${ARCH}"
check_downloader
resolve_install_dir
get_latest_version
download_binary
install_binary
verify

echo -e "\n${BOLD}Done!${RESET} Run:  gitcompare develop master\n"
