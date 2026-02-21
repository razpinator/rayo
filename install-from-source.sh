#!/bin/bash
# Build and install Rayo from source (requires Go).
# Run from repo root: ./install-from-source.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="rayo"
BUILD_DIR="build"
INSTALL_DIR="/usr/local/bin"

echo -e "${YELLOW}Installing Rayo Transpiler from source...${NC}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}Go is not installed.${NC}"
    read -p "Would you like to install Go? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [[ "$OSTYPE" == "darwin"* ]]; then
            if command -v brew &> /dev/null; then
                echo "Installing Go via Homebrew..."
                brew install go
            else
                echo -e "${RED}Error: Homebrew not found. Please install Go manually from https://go.dev/dl/${NC}"
                exit 1
            fi
        elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
            if command -v apt-get &> /dev/null; then
                sudo apt-get update && sudo apt-get install -y golang-go
            elif command -v yum &> /dev/null; then
                sudo yum install -y golang
            elif command -v dnf &> /dev/null; then
                sudo dnf install -y golang
            else
                echo -e "${RED}Error: Package manager not found. Please install Go manually from https://go.dev/dl/${NC}"
                exit 1
            fi
        else
            echo -e "${RED}Error: Unsupported OS. Please install Go manually from https://go.dev/dl/${NC}"
            exit 1
        fi
    else
        echo -e "${RED}Go is required to build Rayo. Exiting.${NC}"
        exit 1
    fi
fi

# Build the binary
echo "Building ${BINARY_NAME}..."
make build

# Check if binary was created
if [ ! -f "${BUILD_DIR}/${BINARY_NAME}" ]; then
    echo -e "${RED}Error: Build failed. Binary not found at ${BUILD_DIR}/${BINARY_NAME}${NC}"
    exit 1
fi

# Check if we have write permissions to install directory
if [ ! -w "${INSTALL_DIR}" ]; then
    echo -e "${YELLOW}Installing to ${INSTALL_DIR} requires sudo privileges...${NC}"
    sudo cp "${BUILD_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/"
else
    cp "${BUILD_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/"
fi

# Verify installation
if command -v "${BINARY_NAME}" &> /dev/null; then
    echo -e "${GREEN}✓ ${BINARY_NAME} installed successfully!${NC}"
    echo "Version: $(${BINARY_NAME} version 2>/dev/null | head -1 || echo 'Unknown')"
    echo "Location: $(which ${BINARY_NAME})"
else
    echo -e "${RED}Error: Installation failed. ${BINARY_NAME} not found in PATH.${NC}"
    exit 1
fi

echo -e "${GREEN}Installation complete!${NC}"
echo "You can now use '${BINARY_NAME}' from anywhere in your terminal."
