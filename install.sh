#!/bin/bash

# R2Go2 Universal Installation Script
# Copyright © 2025 CosmoLabs (https://cosmolabs.org)
# License: MIT

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
REPO="CosmoLabs-org/cosmoflare"
BINARY_NAME="cosmoflare"
INSTALL_DIR="$HOME/.local/bin"
CONFIG_DIR="$HOME/.config/r2go2"

# Emoji for beautiful output
ROCKET="🚀"
CHECK="✅"
WARNING="⚠️"
ERROR="❌"
INFO="ℹ️"
GEAR="⚙️"
DOWNLOAD="📥"
INSTALL="📦"

# Print functions
print_header() {
    echo -e "${CYAN}${ROCKET} R2Go2 Installation Script${NC}"
    echo -e "${CYAN}=====================================${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}${CHECK} $1${NC}"
}

print_error() {
    echo -e "${RED}${ERROR} $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}${WARNING} $1${NC}"
}

print_info() {
    echo -e "${BLUE}${INFO} $1${NC}"
}

print_step() {
    echo -e "${PURPLE}${GEAR} $1${NC}"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Detect OS and architecture
detect_os_arch() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m | tr '[:upper:]' '[:lower:]')

    case $OS in
        darwin)
            OS="darwin"
            ;;
        linux)
            OS="linux"
            ;;
        windows|cygwin|mingw|msys)
            OS="windows"
            ;;
        *)
            print_error "Unsupported operating system: $OS"
            exit 1
            ;;
    esac

    case $ARCH in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        armv7l)
            ARCH="armv7"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    print_info "Detected platform: $OS-$ARCH"
}

# Get latest release version
get_latest_version() {
    print_step "Fetching latest release version..."

    if command_exists curl; then
        VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep -o '"tag_name": "[^"]*' | sed 's/"//g' | sed 's/^v//')
    elif command_exists wget; then
        VERSION=$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" | grep -o '"tag_name": "[^"]*' | sed 's/"//g' | sed 's/^v//')
    else
        print_error "Neither curl nor wget is available"
        exit 1
    fi

    if [ -z "$VERSION" ]; then
        print_error "Failed to fetch latest version"
        exit 1
    fi

    print_success "Latest version: $VERSION"
}

# Download binary
download_binary() {
    print_step "${DOWNLOAD} Downloading R2Go2 binary..."

    FILENAME="${BINARY_NAME}-${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        FILENAME="${FILENAME}.exe"
    fi

    DOWNLOAD_URL="https://github.com/$REPO/releases/download/v$VERSION/$FILENAME"

    if command_exists curl; then
        curl -L -o "$BINARY_NAME" "$DOWNLOAD_URL"
    elif command_exists wget; then
        wget -O "$BINARY_NAME" "$DOWNLOAD_URL"
    else
        print_error "Neither curl nor wget is available"
        exit 1
    fi

    if [ ! -f "$BINARY_NAME" ]; then
        print_error "Failed to download binary"
        exit 1
    fi

    print_success "Downloaded: $BINARY_NAME"
}

# Install binary
install_binary() {
    print_step "${INSTALL} Installing R2Go2..."

    # Create install directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"

    # Make binary executable
    chmod +x "$BINARY_NAME"

    # Move binary to install directory
    mv "$BINARY_NAME" "$INSTALL_DIR/"

    # Add to PATH if not already there
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        SHELL_RC=""
        case $SHELL in
            */bash)
                SHELL_RC="$HOME/.bashrc"
                ;;
            */zsh)
                SHELL_RC="$HOME/.zshrc"
                ;;
            */fish)
                SHELL_RC="$HOME/.config/fish/config.fish"
                ;;
            *)
                print_warning "Unknown shell. Please add $INSTALL_DIR to your PATH manually."
                return
                ;;
        esac

        echo "" >> "$SHELL_RC"
        echo "# R2Go2 binary path" >> "$SHELL_RC"
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_RC"

        print_info "Added $INSTALL_DIR to PATH in $SHELL_RC"
        print_info "Please restart your terminal or run: source $SHELL_RC"
    fi

    print_success "Installed to: $INSTALL_DIR/$BINARY_NAME"
}

# Verify installation
verify_installation() {
    print_step "Verifying installation..."

    if command_exists "$BINARY_NAME"; then
        VERSION_OUTPUT=$("$BINARY_NAME" --version 2>/dev/null || "$BINARY_NAME" version 2>/dev/null || echo "Version $VERSION")
        print_success "Installation verified!"
        print_info "Installed version: $VERSION_OUTPUT"
    else
        print_warning "Binary not found in PATH. You may need to restart your terminal."
        print_info "Binary location: $INSTALL_DIR/$BINARY_NAME"
    fi
}

# Show next steps
show_next_steps() {
    echo ""
    echo -e "${CYAN}🎉 Installation Complete!${NC}"
    echo -e "${CYAN}==========================${NC}"
    echo ""
    print_info "Next steps to get started:"
    echo ""
    echo -e "${BLUE}1. Run the interactive setup:${NC}"
    echo -e "   ${BINARY_NAME} setup"
    echo ""
    echo -e "${BLUE}2. Or set up environment variables:${NC}"
    echo -e "   export CLOUDFLARE_API_TOKEN=\"your_token_here\""
    echo -e "   export CLOUDFLARE_ACCOUNT_ID=\"your_account_id\""
    echo ""
    echo -e "${BLUE}3. Try the TUI dashboard:${NC}"
    echo -e "   ${BINARY_NAME} dashboard"
    echo ""
    echo -e "${BLUE}4. List your buckets:${NC}"
    echo -e "   ${BINARY_NAME} list"
    echo ""
    print_info "📚 Documentation: https://github.com/$REPO"
    print_info "💬 Support: https://github.com/$REPO/issues"
    echo ""
}

# Cleanup on exit
cleanup() {
    if [ -f "$BINARY_NAME" ]; then
        rm -f "$BINARY_NAME"
    fi
}

# Set up cleanup trap
trap cleanup EXIT

# Main installation flow
main() {
    print_header

    # Check prerequisites
    if ! command_exists curl && ! command_exists wget; then
        print_error "Please install curl or wget to download R2Go2"
        exit 1
    fi

    print_info "Starting R2Go2 installation..."
    echo ""

    # Detect platform
    detect_os_arch

    # Get latest version
    get_latest_version

    # Download binary
    download_binary

    # Install binary
    install_binary

    # Verify installation
    verify_installation

    # Show next steps
    show_next_steps
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "R2Go2 Installation Script"
        echo ""
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --version      Show script version"
        echo ""
        echo "This script installs the latest version of R2Go2."
        echo "It will download the binary for your platform and install it to ~/.local/bin"
        echo ""
        exit 0
        ;;
    --version)
        echo "R2Go2 Installation Script v1.0.0"
        exit 0
        ;;
    "")
        # No arguments, run main installation
        main
        ;;
    *)
        print_error "Unknown option: $1"
        print_info "Use --help for usage information"
        exit 1
        ;;
esac