#!/bin/bash

# R2Go2 Local Installation Script
# Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
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
BINARY_NAME="r2go2"
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
    echo -e "${CYAN}${ROCKET} R2Go2 Local Installation Script${NC}"
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

# Use local build
get_local_binary() {
    print_step "Using local build..."

    if [ ! -f "./build/$BINARY_NAME" ]; then
        print_error "Local build not found at ./build/$BINARY_NAME"
        echo -e "${YELLOW}Try running:${NC}"
        echo -e "${YELLOW}  make build${NC}"
        exit 1
    fi

    print_success "Found local build: ./build/$BINARY_NAME"
    VERSION=$(./build/$BINARY_NAME --version 2>/dev/null || echo "unknown")
    print_info "Local version: $VERSION"
}

# Install binary
install_binary() {
    print_step "${INSTALL} Installing R2Go2 locally..."

    # Create install directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"

    # Copy local binary to install directory
    cp "./build/$BINARY_NAME" "$INSTALL_DIR/"

    # Make binary executable
    chmod +x "$INSTALL_DIR/$BINARY_NAME"

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
    print_step "Verifying local installation..."

    if command_exists "$BINARY_NAME"; then
        VERSION_OUTPUT=$("$BINARY_NAME" --version 2>/dev/null || echo "Version unknown")
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
    echo -e "${CYAN}🎉 Local Installation Complete!${NC}"
    echo -e "${CYAN}=============================${NC}"
    echo ""
    print_info "Next steps to get started:"
    echo ""
    echo -e "${BLUE}1. Run the interactive setup:${NC}"
    echo -e "   ${BINARY_NAME} setup"
    echo ""
    echo -e "${BLUE}2. Or set up environment variables:${NC}"
    echo -e '   export CLOUDFLARE_API_TOKEN="your_token_here"'
    echo -e '   export CLOUDFLARE_ACCOUNT_ID="your_account_id"'
    echo ""
    echo -e "${BLUE}3. Try the TUI dashboard:${NC}"
    echo -e "   ${BINARY_NAME} dashboard"
    echo ""
    echo -e "${BLUE}4. List your buckets:${NC}"
    echo -e "   ${BINARY_NAME} list"
    echo ""
    print_info "📚 Documentation: ./GETTING_STARTED.md"
    print_info "💬 Support: https://github.com/$REPO/issues"
    echo ""
}

# Cleanup on exit
cleanup() {
    # No cleanup needed for local install
    return 0
}

# Set up cleanup trap
trap cleanup EXIT

# Main installation flow
main() {
    print_header

    print_info "Starting R2Go2 local installation..."
    echo ""

    # Detect platform
    detect_os_arch

    # Use local build
    get_local_binary

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
        echo "R2Go2 Local Installation Script"
        echo ""
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --version      Show script version"
        echo ""
        echo "This script installs the local build of R2Go2."
        echo "Use this for development testing or when GitHub is unavailable."
        echo ""
        exit 0
        ;;
    --version)
        echo "R2Go2 Local Installation Script v1.0.0"
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