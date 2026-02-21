#!/bin/bash
# Install Rayo from the latest GitHub release (Rust-style one-liner).
# Usage: curl -sSf https://raw.githubusercontent.com/razpinator/rayo/main/install.sh | sh

set -e

REPO="razpinator/rayo"
API_URL="https://api.github.com/repos/${REPO}/releases/latest"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}Installing Rayo...${NC}"

# Detect OS and arch (match GoReleaser asset names)
OS=$(uname -s)
ARCH=$(uname -m)
case "$OS" in
  Darwin)  OS_NAME="Darwin" ;;
  Linux)   OS_NAME="Linux" ;;
  *)
    echo -e "${RED}Unsupported OS: $OS${NC}"
    exit 1
    ;;
esac
case "$ARCH" in
  x86_64)       ARCH_NAME="x86_64" ;;
  amd64)        ARCH_NAME="x86_64" ;;
  arm64|aarch64) ARCH_NAME="arm64" ;;
  *)
    echo -e "${RED}Unsupported arch: $ARCH${NC}"
    exit 1
    ;;
esac

# Fetch latest release and find matching asset
echo "Fetching latest release..."
RELEASE_JSON=$(curl -sSfL "$API_URL")
TAG=$(echo "$RELEASE_JSON" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)
SUFFIX="_${OS_NAME}_${ARCH_NAME}.tar.gz"
DOWNLOAD_URL=$(echo "$RELEASE_JSON" | grep -o "https://[^\"]*${SUFFIX}" | head -1)

if [ -z "$DOWNLOAD_URL" ]; then
  echo -e "${RED}No release found for ${OS_NAME}/${ARCH_NAME}. Check https://github.com/${REPO}/releases${NC}"
  exit 1
fi

TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

echo "Downloading ${TAG}..."
curl -sSfL -o "$TMPDIR/rayo.tar.gz" "$DOWNLOAD_URL"
tar xzf "$TMPDIR/rayo.tar.gz" -C "$TMPDIR"
EXTRACTED=$(find "$TMPDIR" -maxdepth 1 -type d -name 'rayo_*' | head -1)
if [ -z "$EXTRACTED" ]; then
  echo -e "${RED}Download or extract failed.${NC}"
  exit 1
fi

# Prefer ~/.local/bin (no sudo); fall back to /usr/local/bin
INSTALL_DIR="$HOME/.local/bin"
if [ ! -d "$INSTALL_DIR" ]; then
  mkdir -p "$INSTALL_DIR"
fi
if [ ! -w "$INSTALL_DIR" ]; then
  INSTALL_DIR="/usr/local/bin"
  echo -e "${YELLOW}Installing to ${INSTALL_DIR} (sudo required).${NC}"
  sudo cp "$EXTRACTED/rayo" "$EXTRACTED/rayoc" "$INSTALL_DIR/" 2>/dev/null || true
  if [ $? -ne 0 ]; then
    echo -e "${RED}Install failed. Try: sudo cp $EXTRACTED/rayo $EXTRACTED/rayoc /usr/local/bin/${NC}"
    exit 1
  fi
else
  cp "$EXTRACTED/rayo" "$EXTRACTED/rayoc" "$INSTALL_DIR/"
fi

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo -e "${YELLOW}Add to PATH: export PATH=\"\$PATH:$INSTALL_DIR\"${NC}"
  echo "  (add the above to your shell profile for persistence)"
fi

echo -e "${GREEN}✓ Rayo ${TAG} installed to ${INSTALL_DIR}${NC}"
"$INSTALL_DIR/rayo" version 2>/dev/null || true
echo "Run 'rayo version' to verify (ensure $INSTALL_DIR is in your PATH)."
