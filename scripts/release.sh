#!/bin/bash

# Release script for tansive
# This script helps you run goreleaser to build and release your project

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if goreleaser is installed
if ! command -v goreleaser &> /dev/null; then
    print_error "goreleaser is not installed. Please install it first:"
    echo "  go install github.com/goreleaser/goreleaser@latest"
    exit 1
fi

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "Not in a git repository. Please run this script from the project root."
    exit 1
fi

# Check if we have uncommitted changes
if ! git diff-index --quiet HEAD --; then
    print_warning "You have uncommitted changes. Please commit or stash them before releasing."
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if we're on a tag
if ! git describe --exact-match --tags HEAD > /dev/null 2>&1; then
    print_warning "You're not on a git tag. Creating a snapshot release instead."
    SNAPSHOT=true
else
    SNAPSHOT=false
    TAG=$(git describe --exact-match --tags HEAD)
    print_status "Building release for tag: $TAG"
fi

# Set environment variables
export GITHUB_REPOSITORY="tansive/tansive"

# Function to run goreleaser
run_goreleaser() {
    local args="$1"
    
    if [ "$SNAPSHOT" = true ]; then
        print_status "Running goreleaser snapshot..."
        goreleaser release --snapshot --clean $args
    else
        print_status "Running goreleaser release..."
        goreleaser release --clean $args
    fi
}

# Main menu
echo "=== Tansive Release Script ==="
echo "1. Build only (no upload)"
echo "2. Build and upload to GitHub"
echo "3. Build and upload to GitHub (skip validation)"
echo "4. Test configuration"
echo "5. Exit"
echo

read -p "Choose an option (1-5): " -n 1 -r
echo

case $REPLY in
    1)
        print_status "Building binaries and Docker images (no upload)..."
        run_goreleaser "--skip-publish"
        ;;
    2)
        print_status "Building and uploading to GitHub..."
        run_goreleaser ""
        ;;
    3)
        print_status "Building and uploading to GitHub (skip validation)..."
        run_goreleaser "--skip-validate"
        ;;
    4)
        print_status "Testing goreleaser configuration..."
        goreleaser check
        print_status "Configuration is valid!"
        ;;
    5)
        print_status "Exiting..."
        exit 0
        ;;
    *)
        print_error "Invalid option. Please choose 1-5."
        exit 1
        ;;
esac

print_status "Done!" 