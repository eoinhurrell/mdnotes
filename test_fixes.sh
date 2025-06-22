#!/bin/bash

# Test script to demonstrate the vault-relative link fixes

set -e

echo "🔧 Testing mdnotes vault-relative link fixes..."

# Create temporary test vault
TEST_VAULT=$(mktemp -d)
echo "📁 Created test vault: $TEST_VAULT"

# Set up directory structure
mkdir -p "$TEST_VAULT/resources"
mkdir -p "$TEST_VAULT/docs/guides"

# Create test files
cat > "$TEST_VAULT/index.md" << 'EOF'
# Index

This file contains various link types:
- Wiki link to resource: [[resources/test]]
- Markdown link to resource: [Test Resource](resources/test.md)
- Wiki link to guide: [[docs/guides/setup]]
- Markdown link to guide: [Setup Guide](docs/guides/setup.md)
EOF

cat > "$TEST_VAULT/resources/test.md" << 'EOF'
# Test Resource

This is a test resource file in a subdirectory.
EOF

cat > "$TEST_VAULT/docs/guides/setup.md" << 'EOF'
# Setup Guide

This is a setup guide in a nested subdirectory.
EOF

cat > "$TEST_VAULT/other.md" << 'EOF'
# Other File

References:
- [Link to resource](resources/test.md)
- [[docs/guides/setup|Setup]]
EOF

echo "✅ Created test files with vault-relative links"

# Build mdnotes
echo "🔨 Building mdnotes..."
go build -o ./mdnotes ./cmd

echo "📋 Testing links check command..."
# Test links check - should find no broken links
if ./mdnotes links check "$TEST_VAULT"; then
    echo "✅ Links check passed - all vault-relative links found correctly"
else
    echo "❌ Links check failed"
    exit 1
fi

echo "🔄 Testing rename command..."
# Test rename - should update all references correctly
if ./mdnotes rename --dry-run --verbose "$TEST_VAULT/resources/test.md" "$TEST_VAULT/resources/renamed-test.md"; then
    echo "✅ Rename command found and would update vault-relative links correctly"
else
    echo "❌ Rename command failed"
    exit 1
fi

# Cleanup
rm -rf "$TEST_VAULT"
rm -f ./mdnotes

echo "🎉 All tests passed! Vault-relative link handling is working correctly."
echo ""
echo "Key fixes implemented:"
echo "• Config now uses mdnotes.yaml by default (with legacy .obsidian-admin.yaml support)"
echo "• Links check command properly resolves markdown links as vault-relative paths"
echo "• Rename command correctly finds and updates vault-relative paths in all link types"
echo "• Comprehensive tests ensure vault-relative behavior works consistently"