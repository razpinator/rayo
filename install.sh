#!/bin/bash
# Install script for Rayo Transpiler

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="rayoc"
BUILD_DIR="build"
INSTALL_DIR="/usr/local/bin"

echo -e "${YELLOW}Installing Rayo Transpiler...${NC}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed. Please install Go first.${NC}"
    exit 1
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
    echo "Version: $(${BINARY_NAME} --version 2>/dev/null || echo 'Unknown')"
    echo "Location: $(which ${BINARY_NAME})"
else
    echo -e "${RED}Error: Installation failed. ${BINARY_NAME} not found in PATH.${NC}"
    exit 1
fi

echo -e "${GREEN}Installation complete!${NC}"
echo "You can now use '${BINARY_NAME}' from anywhere in your terminal."