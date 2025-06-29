#!/bin/bash

# Release preparation script for mdnotes
# This script validates the codebase and prepares for a release

set -e

VERSION=${1:-}
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v1.2.3"
    exit 1
fi

# Validate version format
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
    echo "Error: Invalid version format: $VERSION"
    echo "Expected format: v1.2.3 or v1.2.3-alpha"
    exit 1
fi

echo "🚀 Preparing release $VERSION"

# Check if we're on the main branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo "❌ Error: Not on main branch (current: $CURRENT_BRANCH)"
    echo "Switch to main branch before creating a release"
    exit 1
fi

# Check for uncommitted changes
if ! git diff --quiet; then
    echo "❌ Error: Uncommitted changes detected"
    echo "Commit or stash changes before creating a release"
    exit 1
fi

# Check if tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo "❌ Error: Tag $VERSION already exists"
    echo "Use a different version or delete the existing tag"
    exit 1
fi

echo "✅ Version validation passed"

# Update dependencies
echo "📦 Updating dependencies..."
go mod tidy
go mod download
go mod verify

# Run comprehensive tests
echo "🧪 Running comprehensive tests..."
go test -race ./...

# Run benchmarks to ensure no performance regressions
echo "⚡ Running benchmarks..."
go test -bench=. -run=^$ ./internal/processor

# Run linting
echo "🔍 Running linter..."
if command -v golangci-lint >/dev/null 2>&1; then
    golangci-lint run --timeout=5m
else
    echo "⚠️  Warning: golangci-lint not found, skipping lint check"
fi

# Security checks
echo "🔒 Running security checks..."

# Run gosec if available
if command -v gosec >/dev/null 2>&1; then
    gosec ./...
else
    echo "⚠️  Warning: gosec not found, skipping security scan"
fi

# Run govulncheck if available
if command -v govulncheck >/dev/null 2>&1; then
    govulncheck ./...
else
    echo "⚠️  Warning: govulncheck not found, installing..."
    go install golang.org/x/vuln/cmd/govulncheck@latest
    govulncheck ./...
fi

# Build and test completions
echo "🏗️  Building and testing completions..."
go build -o mdnotes-temp ./cmd

# Test completion generation
./mdnotes-temp completion bash > /tmp/completion.bash
./mdnotes-temp completion zsh > /tmp/completion.zsh
./mdnotes-temp completion fish > /tmp/completion.fish
./mdnotes-temp completion powershell > /tmp/completion.ps1

# Test completion functionality
echo "🧪 Testing completion functionality..."
if ! ./mdnotes-temp __complete frontmatter ensure --field "" | grep -q "title"; then
    echo "❌ Error: Completion functionality test failed"
    rm -f mdnotes-temp
    exit 1
fi

# Cleanup
rm -f mdnotes-temp /tmp/completion.*

# Test cross-platform builds
echo "🌐 Testing cross-platform builds..."
GOOS=linux GOARCH=amd64 go build -o /tmp/mdnotes-linux ./cmd && rm -f /tmp/mdnotes-linux
GOOS=darwin GOARCH=amd64 go build -o /tmp/mdnotes-darwin ./cmd && rm -f /tmp/mdnotes-darwin
GOOS=windows GOARCH=amd64 go build -o /tmp/mdnotes-windows.exe ./cmd && rm -f /tmp/mdnotes-windows.exe

# Test Docker build
echo "🐳 Testing Docker build..."
if command -v docker >/dev/null 2>&1; then
    docker build -t mdnotes:pre-release .
    docker run --rm mdnotes:pre-release --version
    docker rmi mdnotes:pre-release
else
    echo "⚠️  Warning: Docker not found, skipping Docker build test"
fi

# Check if goreleaser config is valid
echo "📋 Validating GoReleaser configuration..."
if command -v goreleaser >/dev/null 2>&1; then
    goreleaser check
else
    echo "⚠️  Warning: goreleaser not found, skipping config validation"
fi

# Generate changelog preview
echo "📝 Generating changelog preview..."
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [ -n "$LAST_TAG" ]; then
    echo "Changes since $LAST_TAG:"
    git log --oneline "$LAST_TAG"..HEAD
else
    echo "No previous tags found, showing recent commits:"
    git log --oneline -10
fi

echo ""
echo "✅ All pre-release checks passed!"
echo ""
echo "To create the release, run:"
echo "  git tag $VERSION"
echo "  git push origin $VERSION"
echo ""
echo "This will trigger the GitHub Actions release workflow."
echo ""
echo "To create a local test release (without publishing), run:"
echo "  goreleaser release --snapshot --clean"