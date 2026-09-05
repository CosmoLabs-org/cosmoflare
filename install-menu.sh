#!/bin/bash

# R2Go2 Intelligent Menu-Driven Installer
# Copyright © 2025 CosmoLabs (https://cosmolabs.org)
# License: MIT

set -e

# Colors and formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'
UNDERLINE='\033[4m'

# Clear screen function
clear_screen() {
    clear
}

# Header with constant refresh
show_header() {
    clear_screen
    echo -e "${BOLD}${CYAN}"
cat << 'EOF'
┌─────────────────────────────────────────────────────────────────────────┐
│                                                              │
│  🚀 R2Go2 Professional Installation                             │
│  ──────────────────────────────────────────────────────────────────── │
│                                                              │
│  Transform your Cloudflare R2 management experience                     │
│  💫 One command installation • Beautiful terminal dashboard      │
│
└─────────────────────────────────────────────────────────────────┘
EOF
    echo -e "${NC}"
}

# Progress bar
show_progress() {
    local current=$1
    local total=$2
    local width=50
    local percentage=$((current * 100 / total))
    local filled=$((width * current / total))
    local empty=$((width - filled))

    printf "\r${BLUE}[%s%s]${NC}" "$(printf "%*s" $filled | tr ' ' ' ')" "$(printf "%*s" $empty | tr ' ' ' ')"
    printf " ${CYAN}%d%%${NC}" $percentage
}

# Menu option styling
menu_option() {
    local num=$1
    local desc=$2
    local icon=$3
    printf "${BOLD}${BLUE}[%d]${NC} ${icon} ${WHITE}%s${NC}\n" $num "$desc"
}

# Selected menu option
selected_option() {
    local num=$1
    local desc=$2
    local icon=$3
    printf "${BOLD}${GREEN}[%d]${NC} ${icon} ${GREEN}%s${NC} ${DIM}← SELECTED${NC}\n" $num "$desc"
}

# Info message
info_msg() {
    local msg=$1
    echo -e "\n${INFO}ℹ️  $msg${NC}\n"
}

# Success message
success_msg() {
    local msg=$1
    echo -e "\n${GREEN}✅ $msg${NC}\n"
}

# Warning message
warning_msg() {
    local msg=$1
    echo -e "\n${YELLOW}⚠️  $msg${NC}\n"
}

# Error message
error_msg() {
    local msg=$1
    echo -e "\n${RED}❌ $msg${NC}\n"
}

# Get user input
get_input() {
    local prompt=$1
    local default_value=$2
    local result

    if [ -n "$default_value" ]; then
        echo -e "\n${CYAN}🔹 $prompt ${DIM}[$default_value]:${NC} "
    else
        echo -e "\n${CYAN}🔹 $prompt:${NC} "
    fi

    read -r result
    echo "${result:-$default_value}"
}

# Confirm action
confirm_action() {
    local prompt=$1
    local default=${2:-"N"}
    local response

    echo -e "\n${YELLOW}❓ $prompt ${DIM}(y/N) [$default]:${NC} "
    read -r response
    case "$response" in
        [yY]*|*) return 0 ;;
        *) return 1 ;;
    esac
}

# Main menu
show_main_menu() {
    show_header

    echo -e "${BOLD}${WHITE}📋 Installation Menu Options${NC}"
    echo -e "${BOLD}${WHITE}───────────────────────${NC}\n"

    menu_option "1" "Download from GitHub (requires internet)" "🌐"
    menu_option "2" "Use local build (development mode)" "🏗️"
    menu_option "3" "View installation requirements" "📋"
    menu_option "4" "Show installation history" "📜"
    menu_option "5" "Exit installer" "🚪"

    echo -e "${DIM}Enter choice [1-5]:${NC} "
}

# Download from GitHub
download_from_github() {
    show_header
    echo -e "${BOLD}${PURPLE}📥 GitHub Download Mode${NC}"
    echo -e "${BOLD}${PURPLE}──────────────${NC}\n"

    info_msg "Fetching the latest R2Go2 release from GitHub..."

    # Check dependencies
    if ! command_exists curl && ! command_exists wget; then
        error_msg "Both curl and wget are required for download. Please install one first."
        return 1
    fi

    # Get latest version info
    VERSION=$(curl -s "https://api.github.com/repos/CosmoLabs-org/cosmoflare/releases/latest" 2>/dev/null | grep -o '"tag_name": "[^"]*' | sed 's/"//g' | sed 's/^v//' || echo "latest")

    if [ -z "$VERSION" ]; then
        VERSION="latest"
        warning_msg "Could not fetch version, using 'latest'"
    fi

    success_msg "Latest version: $VERSION"

    # Detect platform
    detect_os_arch

    # Download binary
    print_step "Downloading R2Go2 binary..."
    FILENAME="${BINARY_NAME}-${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        FILENAME="${FILENAME}.exe"
    fi

    DOWNLOAD_URL="https://github.com/CosmoLabs-org/cosmoflare/releases/download/v$VERSION/$FILENAME"

    if command_exists curl; then
        curl -L -o "$BINARY_NAME" "$DOWNLOAD_URL" --progress-bar
    else
        wget -O "$BINARY_NAME" "$DOWNLOAD_URL"
    fi

    if [ ! -f "$BINARY_NAME" ]; then
        error_msg "Download failed. Please check your internet connection."
        return 1
    fi

    success_msg "Downloaded: $FILENAME"

    # Continue with installation
    perform_installation
}

# Use local build
use_local_build() {
    show_header
    echo -e "${BOLD}${CYAN}🏗️ Local Build Mode${NC}"
    echo -e "${BOLD}${CYAN}──────────────────${NC}\n"

    info_msg "Using local build from ./build/$BINARY_NAME"

    # Check if local build exists
    if [ ! -f "./build/$BINARY_NAME" ]; then
        error_msg "Local build not found at ./build/$BINARY_NAME"
        echo -e "${YELLOW}Try running: make build${NC}"
        return 1
    fi

    VERSION=$(./build/$BINARY_NAME --version 2>/dev/null || echo "unknown")
    success_msg "Found local build: v$VERSION"

    # Continue with installation
    perform_installation
}

# Perform installation
perform_installation() {
    show_header
    echo -e "${BOLD}${GREEN}📦 Installation Process${NC}"
    echo -e "${BOLD}${GREEN}───────────────────${NC}\n"

    # Progress tracking
    echo -e "${DIM}Step 1/5: Preparing installation environment...${NC}"
    show_progress 1 5
    sleep 1

    echo -e "${DIM}Step 2/5: Creating directories...${NC}"
    show_progress 2 5
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"
    sleep 1

    echo -e "${DIM}Step 3/5: Installing binary...${NC}"
    show_progress 3 5

    if [ -f "$BINARY_NAME" ]; then
        mv "$BINARY_NAME" "$INSTALL_DIR/"
    elif [ -f "./build/$BINARY_NAME" ]; then
        cp "./build/$BINARY_NAME" "$INSTALL_DIR/"
    else
        error_msg "Binary not found. Please ensure you've built or downloaded R2Go2."
        return 1
    fi
    sleep 1

    echo -e "${DIM}Step 4/5: Setting up permissions...${NC}"
    show_progress 4 5
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    sleep 1

    echo -e "${DIM}Step 5/5: Configuring PATH...${NC}"
    show_progress 5 5

    # Add to PATH if needed
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
                SHELL_RC="$HOME/.profile"
                ;;
        esac

        if ! grep -q "$INSTALL_DIR" "$SHELL_RC" 2>/dev/null; then
            echo "" >> "$SHELL_RC"
            echo "# R2Go2 binary path" >> "$SHELL_RC"
            echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_RC"
            success_msg "Added to PATH in $SHELL_RC"
            info_msg "Run 'source $SHELL_RC' to apply changes"
        else
            success_msg "PATH already configured"
        fi
    else
        success_msg "PATH already contains $INSTALL_DIR"
    fi

    sleep 1

    success_msg "Installation completed successfully!"
}

# Show requirements
show_requirements() {
    show_header
    echo -e "${BOLD}${WHITE}📋 Installation Requirements${NC}"
    echo -e "${BOLD}${WHITE}──────────────────${NC}\n"

    echo -e "${CYAN}System Requirements:${NC}"
    echo -e "  • Unix-like OS (macOS, Linux, WSL)${NC}"
    echo -e "  • Command-line interface (Terminal.app, iTerm2, etc.)${NC}"
    echo -e "  • curl or wget (for GitHub downloads)${NC}"
    echo -e ""

    echo -e "${CYAN}Optional (for GitHub downloads):${NC}"
    echo -e "  • Internet connection${NC}"
    echo -e "  • GitHub account${NC}"
    echo -e ""

    echo -e "${CYAN}For local builds:${NC}"
    echo -e "  • Go 1.21+ installed${NC}"
    echo -e "  • Built R2Go2 binary in ./build/${NC}"
    echo -e ""

    echo -e "${CYAN}After installation:${NC}"
    echo -e "  • Cloudflare account with R2 enabled${NC}"
    echo -e "  • API token with appropriate permissions${NC}"
    echo -e "  • Account ID from Cloudflare dashboard${NC}"
    echo ""

    echo -e "${YELLOW}💡 Tip: Run './r2go2 setup' after installation for guided setup!${NC}\n"

    echo -e "${BOLD}Press any key to return to main menu${NC}"
    read -r
}

# Show installation history
show_history() {
    show_header
    echo -e "${BOLD}${WHITE}📜 Installation History${NC}"
    echo -e "${BOLD}${WHITE}───────────────${NC}\n"

    echo -e "${CYAN}Recent Installation Sessions:${NC}"
    echo ""

    if [ -f "$CONFIG_DIR/install.log" ]; then
        tail -10 "$CONFIG_DIR/install.log" | while read line; do
            echo "  $line"
        done
    else
        echo -e "${DIM}No installation history found.${NC}"
    fi

    echo -e "${BOLD}${WHITE}Current Version:${NC}"
    if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        VERSION=$("$INSTALL_DIR/$BINARY_NAME" --version 2>/dev/null || echo "unknown")
        echo -e "  $VERSION - Located at $INSTALL_DIR/$BINARY_NAME${NC}"
    else
        echo -e "  Not installed${NC}"
    fi

    echo ""
    echo -e "${BOLD}${WHITE}Configuration:${NC}"
    if [ -f "$CONFIG_DIR/config.json" ]; then
        echo -e "  Configuration found at $CONFIG_DIR/config.json${NC}"
    else
        echo -  No configuration found${NC}
    fi

    echo ""
    echo -e "${BOLD}${WHITE}Press any key to return to main menu${NC}"
    read -r
}

# Exit installer
exit_installer() {
    show_header
    echo -e "${BOLD}${RED}🚪 Exiting Installation${NC}"
    echo -e "${BOLD}${RED}──────────────────${NC}\n"

    echo -e "${DIM}Thank you for trying R2Go2!${NC}"
    echo -e "${DIM}Visit: https://github.com/CosmoLabs-org/cosmoflare${NC}"
    echo -e "${DIM}Need help? Open an issue on GitHub!${NC}"
    echo ""
    exit 0
}

# Main installer loop
main_installer() {
    while true; do
        show_main_menu
        read -r choice
        case "$choice" in
            1)
                download_from_github
                ;;
            2)
                use_local_build
                ;;
            3)
                show_requirements
                ;;
            4)
                show_history
                ;;
            5)
                exit_installer
                ;;
            *)
                warning_msg "Invalid choice. Please select 1-5."
                ;;
        esac

        if [ "$choice" = "5" ]; then
            break
        fi
        echo -e "${NC}"
        sleep 1
    done
}

# Print step with styling
print_step() {
    local msg=$1
    echo -e "${BOLD}${CYAN}⚙️  $msg${NC}"
}

# Initialize
initialize() {
    # Set up signal handling
    trap 'echo -e "\n${YELLOW}⚠️  Installation interrupted${NC}"; exit 1' INT

    # Binary and installation configuration
    BINARY_NAME="cosmoflare"
    INSTALL_DIR="$HOME/.local/bin"
    CONFIG_DIR="$HOME/.config/r2go2"

    # Check if running in interactive terminal
    if [ -t 0 ]; then
        # Interactive mode - show colors
        :
    else
        # Non-interactive mode - remove color codes
        RED=''
        GREEN=''
        YELLOW=''
        BLUE=''
        PURPLE=''
        CYAN=''
        WHITE=''
        BOLD=''
        DIM=''
        UNDERLINE=''
        NC=''
    fi
}

# Error handling
handle_error() {
    local error_msg=$1
    echo -e "\n${RED}❌ Error: $error_msg${NC}" >&2
    exit 1
}

# Main execution
main() {
    initialize

    # Check dependencies
    if [ ! -f "./build/R2Go2" ] && ! command_exists curl && ! command_exists wget; then
        handle_error "Neither curl nor wget available for downloading."
    fi

    main_installer
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi