#!/usr/bin/env bash
set -euo pipefail

APP="openclaw-whatsapp"
REPO="cyberfront-ai/openclaw-whatsapp"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)             echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Determine version
if [ -n "${1:-}" ]; then
  VERSION="$1"
else
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)"
fi
echo "Installing ${APP} ${VERSION} (${OS}/${ARCH})..."

BINARY="${APP}-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}"

# Install location
if [ -w /usr/local/bin ]; then
  INSTALL_DIR="/usr/local/bin"
else
  INSTALL_DIR="${HOME}/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi

curl -fsSL "$URL" -o "${INSTALL_DIR}/${APP}"
chmod +x "${INSTALL_DIR}/${APP}"

echo "Installed ${APP} to ${INSTALL_DIR}/${APP}"
echo "Run: ${APP} start"
