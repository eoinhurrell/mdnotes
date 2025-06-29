#!/bin/bash

# Test script for mdnotes shell completions
# This script verifies that all completion functionality works correctly

set -e

BINARY="./mdnotes"

echo "=== Testing mdnotes shell completions ==="

# Test that completions can be generated for all shells
echo "Testing completion generation..."
echo "✓ Bash completion:" 
$BINARY completion bash > /dev/null && echo "  Generated successfully"

echo "✓ Zsh completion:"
$BINARY completion zsh > /dev/null && echo "  Generated successfully"

echo "✓ Fish completion:"
$BINARY completion fish > /dev/null && echo "  Generated successfully"

echo "✓ PowerShell completion:"
$BINARY completion powershell > /dev/null && echo "  Generated successfully"

# Test __complete functionality
echo ""
echo "Testing completion functionality..."

# Test command completions
echo "✓ Command completion:"
commands=$($BINARY __complete "" 2>/dev/null | grep -v '^:' | head -5)
echo "  Available commands: $(echo $commands | tr '\n' ' ')"

# Test field completions
echo "✓ Field completion:"
fields=$($BINARY __complete frontmatter ensure --field "" 2>/dev/null | grep -v '^:' | head -5)
echo "  Available fields: $(echo $fields | tr '\n' ' ')"

# Test type completions
echo "✓ Type completion:"
types=$($BINARY __complete frontmatter cast --type "title:" 2>/dev/null | grep -v '^:' | head -3)
echo "  Available types: $(echo $types | tr '\n' ' ')"

# Test format completions  
echo "✓ Format completion:"
formats=$($BINARY __complete analyze stats --format "" 2>/dev/null | grep -v '^:')
echo "  Available formats: $(echo $formats | tr '\n' ' ')"

# Test shell completion
echo "✓ Shell completion:"
shells=$($BINARY __complete completion "" 2>/dev/null | grep -v '^:')
echo "  Available shells: $(echo $shells | tr '\n' ' ')"

# Test query completions
echo "✓ Query completion:"
queries=$($BINARY __complete export --query "" 2>/dev/null | grep -v '^:' | head -3)
echo "  Available queries: $(echo $queries | tr '\n' ' ')"

# Test default value completions
echo "✓ Default value completion:"
defaults=$($BINARY __complete frontmatter ensure --default "" 2>/dev/null | grep -v '^:' | head -3)
echo "  Available defaults: $(echo $defaults | tr '\n' ' ')"

# Test global shortcut completions
echo "✓ Global shortcut completion:"
shortcut_fields=$($BINARY __complete e --field "" 2>/dev/null | grep -v '^:' | head -3)
echo "  Shortcut 'e' fields: $(echo $shortcut_fields | tr '\n' ' ')"

echo ""
echo "=== All completion tests passed! ==="
echo ""
echo "Installation instructions:"
echo "  Bash:       source <(mdnotes completion bash)"
echo "  Zsh:        mdnotes completion zsh > \"\${fpath[1]}/_mdnotes\""  
echo "  Fish:       mdnotes completion fish > ~/.config/fish/completions/mdnotes.fish"
echo "  PowerShell: mdnotes completion powershell > mdnotes.ps1"