#!/bin/bash
# Installation script for Git hooks
# This script installs pre-commit and pre-push hooks for the Gor Framework

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

HOOKS_DIR=".git/hooks"
SCRIPTS_DIR="scripts/git-hooks"

echo -e "${BLUE}🔧 Installing Git hooks for Gor Framework...${NC}"
echo ""

# Check if we're in a git repository
if [ ! -d ".git" ]; then
    echo -e "${RED}❌ Error: Not in a git repository root${NC}"
    echo "Please run this script from the repository root directory"
    exit 1
fi

# Check if hooks directory exists
if [ ! -d "$SCRIPTS_DIR" ]; then
    echo -e "${RED}❌ Error: $SCRIPTS_DIR directory not found${NC}"
    exit 1
fi

# Function to install a hook
install_hook() {
    local hook_name=$1
    local source_file="$SCRIPTS_DIR/$hook_name"
    local target_file="$HOOKS_DIR/$hook_name"

    if [ ! -f "$source_file" ]; then
        echo -e "${YELLOW}⚠️  Warning: $source_file not found, skipping${NC}"
        return
    fi

    # Backup existing hook if it exists
    if [ -f "$target_file" ]; then
        echo -e "${YELLOW}📦 Backing up existing $hook_name to $hook_name.backup${NC}"
        cp "$target_file" "$target_file.backup"
    fi

    # Copy and make executable
    cp "$source_file" "$target_file"
    chmod +x "$target_file"

    echo -e "${GREEN}✅ Installed $hook_name hook${NC}"
}

# Install hooks
install_hook "pre-commit"
install_hook "pre-push"

echo ""
echo -e "${GREEN}🎉 Git hooks installed successfully!${NC}"
echo ""
echo "Hooks installed:"
echo "  • pre-commit: Fast checks (formatting, vet, compile)"
echo "  • pre-push:   Comprehensive checks (tests, linting, race detection)"
echo ""
echo "To bypass hooks in emergency (use sparingly):"
echo "  git commit --no-verify"
echo "  git push --no-verify"
echo ""

# Check for required tools and provide installation instructions
echo -e "${BLUE}Checking for required tools...${NC}"

# Check for Go
if ! command -v go >/dev/null 2>&1; then
    echo -e "${RED}❌ Go is not installed${NC}"
    exit 1
else
    echo -e "${GREEN}✅ Go is installed ($(go version | awk '{print $3}'))${NC}"
fi

# Check for golangci-lint (recommended)
if ! command -v golangci-lint >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  golangci-lint is not installed (recommended)${NC}"
    echo "   Install with:"
    echo "   go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2"
else
    echo -e "${GREEN}✅ golangci-lint is installed ($(golangci-lint version 2>/dev/null | head -1))${NC}"
fi

# Check for gosec (optional)
if ! command -v gosec >/dev/null 2>&1; then
    echo -e "${YELLOW}ℹ️  gosec is not installed (optional)${NC}"
    echo "   Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"
else
    echo -e "${GREEN}✅ gosec is installed${NC}"
fi

echo ""
echo -e "${BLUE}💡 Tip: Run 'make help' to see all available commands${NC}"