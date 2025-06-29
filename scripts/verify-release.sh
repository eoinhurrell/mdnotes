#!/bin/bash

# Release verification script for mdnotes
# This script verifies a release after it has been published

set -e

VERSION=${1:-}
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v1.2.3"
    exit 1
fi

REPO_OWNER="eoinhurrell"
REPO_NAME="mdnotes"
GITHUB_API="https://api.github.com/repos/$REPO_OWNER/$REPO_NAME"

echo "🔍 Verifying release $VERSION"

# Check if GitHub CLI is available
if ! command -v gh >/dev/null 2>&1; then
    echo "⚠️  Warning: GitHub CLI (gh) not found. Some verifications will be skipped."
    USE_GH=false
else
    USE_GH=true
fi

# Get release information
echo "📋 Fetching release information..."
if [ "$USE_GH" = true ]; then
    RELEASE_JSON=$(gh api repos/$REPO_OWNER/$REPO_NAME/releases/tags/$VERSION 2>/dev/null || echo "")
    if [ -z "$RELEASE_JSON" ]; then
        echo "❌ Error: Release $VERSION not found"
        exit 1
    fi
    
    RELEASE_ID=$(echo "$RELEASE_JSON" | jq -r '.id')
    RELEASE_NAME=$(echo "$RELEASE_JSON" | jq -r '.name')
    IS_PRERELEASE=$(echo "$RELEASE_JSON" | jq -r '.prerelease')
    
    echo "✅ Found release: $RELEASE_NAME (ID: $RELEASE_ID, Prerelease: $IS_PRERELEASE)"
else
    echo "⚠️  Skipping release information fetch (gh CLI not available)"
fi

# List expected assets
EXPECTED_ASSETS=(
    "checksums.txt"
    "mdnotes_Linux_x86_64.tar.gz"
    "mdnotes_Linux_arm64.tar.gz"
    "mdnotes_Darwin_x86_64.tar.gz"
    "mdnotes_Darwin_arm64.tar.gz"
    "mdnotes_Windows_x86_64.zip"
    "mdnotes_${VERSION#v}_sbom.spdx.json"
)

echo "🗂️  Checking release assets..."
if [ "$USE_GH" = true ]; then
    ASSETS=$(echo "$RELEASE_JSON" | jq -r '.assets[].name')
    
    MISSING_ASSETS=()
    for asset in "${EXPECTED_ASSETS[@]}"; do
        if ! echo "$ASSETS" | grep -q "^$asset$"; then
            MISSING_ASSETS+=("$asset")
        else
            echo "✅ Found: $asset"
        fi
    done
    
    if [ ${#MISSING_ASSETS[@]} -gt 0 ]; then
        echo "❌ Missing assets:"
        printf '  - %s\n' "${MISSING_ASSETS[@]}"
        exit 1
    fi
    
    echo "✅ All expected assets found"
else
    echo "⚠️  Skipping asset verification (gh CLI not available)"
fi

# Test binary downloads and functionality
echo "⬇️  Testing binary downloads..."
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

if [ "$USE_GH" = true ]; then
    # Download and test Linux binary
    echo "Testing Linux binary..."
    gh release download "$VERSION" -R "$REPO_OWNER/$REPO_NAME" -p "*Linux_x86_64*"
    tar -xzf mdnotes_Linux_x86_64.tar.gz
    
    # Basic functionality test
    if ./mdnotes --version | grep -q "$VERSION"; then
        echo "✅ Linux binary version check passed"
    else
        echo "❌ Linux binary version check failed"
        cd - && rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    # Test help output
    if ./mdnotes --help | grep -q "CLI tool for managing Obsidian"; then
        echo "✅ Linux binary help output check passed"
    else
        echo "❌ Linux binary help output check failed"
        cd - && rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    # Test completion generation
    if ./mdnotes completion bash | grep -q "mdnotes"; then
        echo "✅ Linux binary completion generation check passed"
    else
        echo "❌ Linux binary completion generation check failed"
        cd - && rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    echo "✅ Linux binary functionality verified"
else
    echo "⚠️  Skipping binary download test (gh CLI not available)"
fi

cd - && rm -rf "$TEMP_DIR"

# Test Docker images
echo "🐳 Testing Docker images..."
if command -v docker >/dev/null 2>&1; then
    # Test versioned image
    if docker run --rm "ghcr.io/$REPO_OWNER/$REPO_NAME:$VERSION" --version | grep -q "$VERSION"; then
        echo "✅ Versioned Docker image test passed"
    else
        echo "❌ Versioned Docker image test failed"
        exit 1
    fi
    
    # Test latest image
    if docker run --rm "ghcr.io/$REPO_OWNER/$REPO_NAME:latest" --version; then
        echo "✅ Latest Docker image test passed"
    else
        echo "❌ Latest Docker image test failed"
        exit 1
    fi
    
    echo "✅ Docker images verified"
else
    echo "⚠️  Warning: Docker not found, skipping Docker image tests"
fi

# Verify checksums
echo "🔐 Verifying checksums..."
if [ "$USE_GH" = true ]; then
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    
    # Download checksums file and a binary to verify
    gh release download "$VERSION" -R "$REPO_OWNER/$REPO_NAME" -p "checksums.txt"
    gh release download "$VERSION" -R "$REPO_OWNER/$REPO_NAME" -p "*Linux_x86_64*"
    
    # Verify checksum
    if sha256sum -c checksums.txt --ignore-missing; then
        echo "✅ Checksum verification passed"
    else
        echo "❌ Checksum verification failed"
        cd - && rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    cd - && rm -rf "$TEMP_DIR"
else
    echo "⚠️  Skipping checksum verification (gh CLI not available)"
fi

# Check SBOM
echo "📋 Checking SBOM..."
if [ "$USE_GH" = true ]; then
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    
    # Download SBOM
    gh release download "$VERSION" -R "$REPO_OWNER/$REPO_NAME" -p "*sbom*"
    
    # Basic SBOM validation
    if [ -f "mdnotes_${VERSION#v}_sbom.spdx.json" ]; then
        if jq -e '.spdxVersion' "mdnotes_${VERSION#v}_sbom.spdx.json" >/dev/null; then
            echo "✅ SBOM format validation passed"
        else
            echo "❌ SBOM format validation failed"
            cd - && rm -rf "$TEMP_DIR"
            exit 1
        fi
    else
        echo "❌ SBOM file not found"
        cd - && rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    cd - && rm -rf "$TEMP_DIR"
else
    echo "⚠️  Skipping SBOM verification (gh CLI not available)"
fi

# Test Homebrew formula (if available)
echo "🍺 Testing Homebrew formula..."
if command -v brew >/dev/null 2>&1; then
    # Note: This assumes you have a homebrew tap set up
    if brew search "$REPO_OWNER/tap/$REPO_NAME" >/dev/null 2>&1; then
        echo "✅ Homebrew formula found"
        # You could add more specific tests here
    else
        echo "⚠️  Homebrew formula not found (may not be published yet)"
    fi
else
    echo "⚠️  Homebrew not found, skipping formula test"
fi

echo ""
echo "🎉 Release verification completed successfully!"
echo ""
echo "Release $VERSION appears to be working correctly."
echo ""
echo "Manual verification checklist:"
echo "  □ Test installation on different platforms"
echo "  □ Verify shell completions work in actual shells"
echo "  □ Test key functionality with real Obsidian vaults"
echo "  □ Check release notes for accuracy"
echo "  □ Verify all documentation links work"
echo ""
echo "If this is a public release, consider:"
echo "  □ Announcing on relevant channels"
echo "  □ Updating project documentation"
echo "  □ Notifying users of breaking changes (if any)"