# mdnotes User Guide

mdnotes is a powerful CLI tool for managing Obsidian markdown note vaults. It provides automated batch operations for frontmatter management, heading fixes, link conversions, and external service integrations.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Commands Reference](#commands-reference)
- [Configuration](#configuration)
- [Common Workflows](#common-workflows)
- [Troubleshooting](#troubleshooting)

## Installation

### From Source

```bash
git clone https://github.com/eoinhurrell/mdnotes.git
cd mdnotes
go build -o mdnotes ./cmd
sudo mv mdnotes /usr/local/bin/
```

### Verify Installation

```bash
mdnotes --version
mdnotes --help
```

## Quick Start

### Basic Operations

1. **Navigate to your vault:**
   ```bash
   cd /path/to/your/obsidian-vault
   ```

2. **Ensure all notes have required frontmatter fields:**
   ```bash
   mdnotes frontmatter ensure --field tags --default "[]" .
   ```

3. **Validate frontmatter consistency:**
   ```bash
   mdnotes frontmatter validate --required title --required tags .
   ```

4. **Analyze vault health:**
   ```bash
   mdnotes analyze health .
   ```

### Preview Changes (Dry Run)

Always test changes with `--dry-run` first:

```bash
mdnotes frontmatter ensure --field created --default "{{current_date}}" --dry-run .
```

## Core Concepts

### Vault Structure

mdnotes expects an Obsidian-style vault structure:
```
vault/
├── .obsidian/          # Obsidian configuration (ignored)
├── note1.md
├── note2.md
├── folder/
│   ├── note3.md
│   └── note4.md
└── templates/
    └── daily.md
```

### Frontmatter

mdnotes processes YAML frontmatter in markdown files:

```markdown
---
title: My Note
tags: [personal, important]
created: 2023-01-15
priority: 5
published: false
---

# My Note

Content goes here...
```

### Template Variables

Use template variables for dynamic values:

- `{{current_date}}` - Current date (YYYY-MM-DD)
- `{{current_datetime}}` - Current datetime (ISO format)
- `{{filename}}` - Base filename without extension
- `{{title}}` - Value from title frontmatter field
- `{{file_mtime}}` - File modification date
- `{{relative_path}}` - Path relative to vault root
- `{{parent_dir}}` - Parent directory name
- `{{uuid}}` - Random UUID v4

### Template Filters

Apply transformations with pipe syntax:

- `{{filename|upper}}` - UPPERCASE
- `{{filename|lower}}` - lowercase
- `{{title|slug}}` - URL-friendly slug
- `{{file_mtime|date:Jan 2, 2006}}` - Custom date format

## Commands Reference

### Frontmatter Management

#### Ensure Fields
Add missing frontmatter fields with default values:

```bash
# Basic field with static default
mdnotes frontmatter ensure --field tags --default "[]" /path/to/vault

# Dynamic defaults with templates
mdnotes frontmatter ensure --field created --default "{{current_date}}" /path/to/vault

# Multiple fields
mdnotes frontmatter ensure \
  --field id --default "{{filename|slug}}" \
  --field modified --default "{{file_mtime}}" \
  /path/to/vault
```

#### Validate Fields
Check for required fields and correct types:

```bash
# Check required fields
mdnotes frontmatter validate --required title --required tags /path/to/vault

# Validate field types
mdnotes frontmatter validate \
  --type tags:array \
  --type priority:number \
  --type published:boolean \
  /path/to/vault

# Verbose output
mdnotes frontmatter validate --required title --verbose /path/to/vault
```

#### Type Casting
Convert field values to proper types:

```bash
# Auto-detect and cast all fields
mdnotes frontmatter cast --auto-detect /path/to/vault

# Cast specific fields
mdnotes frontmatter cast \
  --field created --type date \
  --field priority --type number \
  /path/to/vault

# Preview changes
mdnotes frontmatter cast --auto-detect --dry-run /path/to/vault
```

#### Sync with File System
Synchronize frontmatter with file system metadata:

```bash
# Sync modification time
mdnotes frontmatter sync --field modified --source file-mtime /path/to/vault

# Extract from filename patterns
mdnotes frontmatter sync \
  --field date \
  --source "filename:pattern:^(\\d{8})" \
  /path/to/vault

# Sync directory structure
mdnotes frontmatter sync --field category --source "path:dir" /path/to/vault
```

### Heading Management

#### Analyze Headings
Check heading structure and identify issues:

```bash
# Analyze all files
mdnotes headings analyze /path/to/vault

# Verbose output with suggestions
mdnotes headings analyze --verbose /path/to/vault
```

#### Fix Headings
Automatically fix heading structure issues:

```bash
# Ensure H1 matches title
mdnotes headings fix --ensure-h1-title /path/to/vault

# Fix multiple H1s
mdnotes headings fix --single-h1 /path/to/vault

# Fix heading sequence
mdnotes headings fix --fix-sequence /path/to/vault

# All fixes
mdnotes headings fix --ensure-h1-title --single-h1 --fix-sequence /path/to/vault

# Preview changes
mdnotes headings fix --ensure-h1-title --dry-run /path/to/vault
```

### Link Management

#### Check Links
Verify internal link integrity:

```bash
# Check for broken links
mdnotes links check /path/to/vault

# Show broken links with suggestions
mdnotes links check --verbose /path/to/vault
```

#### Convert Links
Convert between wiki links and markdown links:

```bash
# Convert wiki links to markdown
mdnotes links convert --from wiki --to markdown /path/to/vault

# Convert markdown links to wiki
mdnotes links convert --from markdown --to wiki /path/to/vault

# Preview conversions
mdnotes links convert --from wiki --to markdown --dry-run /path/to/vault
```

### Vault Analysis

#### Statistics
Generate comprehensive vault statistics:

```bash
# Basic statistics
mdnotes analyze stats /path/to/vault

# JSON output
mdnotes analyze stats --format json /path/to/vault

# Save to file
mdnotes analyze stats --output vault-stats.txt /path/to/vault
```

#### Duplicates
Find duplicate content:

```bash
# Find duplicates
mdnotes analyze duplicates /path/to/vault

# Adjust similarity threshold
mdnotes analyze duplicates --similarity 0.9 /path/to/vault

# JSON output
mdnotes analyze duplicates --format json /path/to/vault
```

#### Health Check
Assess overall vault health:

```bash
# Health report
mdnotes analyze health /path/to/vault

# JSON output for scripting
mdnotes analyze health --format json /path/to/vault
```

### Batch Operations

#### Execute Batch
Run multiple operations from configuration:

```bash
# Execute from config file
mdnotes batch execute --config batch-config.yaml /path/to/vault

# With progress reporting
mdnotes batch execute \
  --config batch-config.yaml \
  --progress terminal \
  /path/to/vault

# Parallel processing
mdnotes batch execute \
  --config batch-config.yaml \
  --workers 8 \
  /path/to/vault

# Dry run
mdnotes batch execute \
  --config batch-config.yaml \
  --dry-run \
  /path/to/vault
```

#### Validate Configuration
Check batch configuration files:

```bash
# Validate config
mdnotes batch validate batch-config.yaml

# Detailed validation
mdnotes batch validate --verbose batch-config.yaml
```

## Configuration

### Configuration File

Create `.obsidian-admin.yaml` in your vault root:

```yaml
version: "1.0"

vault:
  path: "."
  ignore_patterns:
    - ".obsidian/*"
    - "*.tmp"
    - "*.bak"
    - ".DS_Store"

frontmatter:
  required_fields:
    - "title"
    - "tags"
    - "created"
  type_rules:
    fields:
      created: date
      modified: date
      tags: array
      priority: number
      published: boolean

linkding:
  api_url: "${LINKDING_URL}"
  api_token: "${LINKDING_TOKEN}"
  sync_title: true
  sync_tags: true

batch:
  stop_on_error: false
  create_backup: true
  max_workers: 4

safety:
  backup_retention: "24h"
  max_backups: 50
```

### Environment Variables

Set sensitive values via environment:

```bash
export LINKDING_URL="https://linkding.example.com"
export LINKDING_TOKEN="your-api-token"
```

### Configuration Locations

mdnotes searches for configuration in order:

1. `--config` flag
2. `.obsidian-admin.yaml` (current directory)
3. `obsidian-admin.yaml` (current directory)
4. `~/.config/obsidian-admin/config.yaml`
5. `~/.obsidian-admin.yaml`
6. `/etc/obsidian-admin/config.yaml`

## Common Workflows

### Daily Maintenance

```bash
#!/bin/bash
# daily-vault-maintenance.sh

VAULT_PATH="/path/to/vault"

# 1. Ensure consistent frontmatter
mdnotes frontmatter ensure \
  --field created --default "{{current_date}}" \
  --field modified --default "{{file_mtime}}" \
  "$VAULT_PATH"

# 2. Validate all fields
mdnotes frontmatter validate \
  --required title \
  --required tags \
  --type created:date \
  "$VAULT_PATH"

# 3. Fix heading issues
mdnotes headings fix --ensure-h1-title "$VAULT_PATH"

# 4. Check link integrity
mdnotes links check "$VAULT_PATH"

# 5. Generate health report
mdnotes analyze health "$VAULT_PATH"
```

### New Vault Setup

```bash
#!/bin/bash
# setup-new-vault.sh

VAULT_PATH="$1"

if [ -z "$VAULT_PATH" ]; then
  echo "Usage: $0 /path/to/vault"
  exit 1
fi

cd "$VAULT_PATH"

# 1. Create configuration
cat > .obsidian-admin.yaml << EOF
version: "1.0"
vault:
  ignore_patterns:
    - ".obsidian/*"
    - "*.tmp"
frontmatter:
  required_fields:
    - "title"
    - "tags"
    - "created"
  type_rules:
    fields:
      created: date
      tags: array
EOF

# 2. Add missing frontmatter to all files
mdnotes frontmatter ensure \
  --field title --default "{{filename}}" \
  --field tags --default "[]" \
  --field created --default "{{current_date}}" \
  .

# 3. Validate results
mdnotes frontmatter validate --required title --required tags .

echo "Vault setup complete!"
```

### Bulk Import Processing

```bash
#!/bin/bash
# process-bulk-import.sh

IMPORT_DIR="$1"

# 1. Preview what will be changed
mdnotes frontmatter ensure \
  --field imported --default "{{current_date}}" \
  --field source --default "bulk-import" \
  --dry-run \
  "$IMPORT_DIR"

# 2. Apply changes if preview looks good
read -p "Apply changes? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  mdnotes frontmatter ensure \
    --field imported --default "{{current_date}}" \
    --field source --default "bulk-import" \
    "$IMPORT_DIR"
  
  # 3. Type casting for consistency
  mdnotes frontmatter cast --auto-detect "$IMPORT_DIR"
  
  # 4. Generate import report
  mdnotes analyze stats --output import-report.txt "$IMPORT_DIR"
fi
```

### Link Format Migration

```bash
#!/bin/bash
# migrate-to-markdown-links.sh

VAULT_PATH="$1"

# 1. Preview conversion
mdnotes links convert \
  --from wiki \
  --to markdown \
  --dry-run \
  "$VAULT_PATH"

# 2. Create backup
cp -r "$VAULT_PATH" "${VAULT_PATH}.backup"

# 3. Convert links
mdnotes links convert \
  --from wiki \
  --to markdown \
  "$VAULT_PATH"

# 4. Verify no broken links
mdnotes links check "$VAULT_PATH"
```

## Troubleshooting

### Common Issues

#### Permission Errors
```bash
Error: permission denied accessing file: /path/to/file.md

Suggestion: Check that you have read/write permissions for this file and its parent directory.
```

**Solution:** Check file permissions and ownership:
```bash
ls -la /path/to/file.md
chmod 644 /path/to/file.md  # If needed
```

#### Invalid YAML Syntax
```bash
Error: syntax error in file note.md at line 3: yaml: unmarshal errors

Suggestion: Check the YAML syntax in your frontmatter.
```

**Solution:** Validate YAML syntax:
- Check indentation (use spaces, not tabs)
- Quote special characters
- Ensure list syntax is correct

#### Type Validation Errors
```bash
Error: invalid type for field 'created': expected date, got '2023/01/15'

Suggestion: Date must be in YYYY-MM-DD format.
```

**Solution:** Use correct format:
```yaml
created: 2023-01-15  # Correct
created: "2023/01/15"  # Incorrect
```

#### Configuration Issues
```bash
Error: configuration error: unknown field 'invalid_field'

Suggestion: Check your configuration file for syntax errors.
```

**Solution:** Validate configuration:
```bash
mdnotes batch validate .obsidian-admin.yaml
```

### Performance Tips

#### Large Vaults (5000+ files)
- Use `--workers` flag for parallel processing
- Enable chunked processing in configuration
- Use `--quiet` for faster execution
- Consider processing subdirectories separately

#### Memory Usage
- Monitor with `--verbose` flag
- Use streaming mode for very large vaults
- Process in smaller batches if memory issues occur

#### Network Operations
- Set appropriate timeouts in configuration
- Use retry logic for unreliable connections
- Cache results when possible

### Debug Mode

Enable verbose output for troubleshooting:

```bash
# Verbose output
mdnotes --verbose frontmatter ensure --field tags --default "[]" .

# Debug configuration loading
mdnotes --verbose analyze stats .

# See detailed error information
mdnotes --verbose batch execute --config config.yaml .
```

### Getting Help

1. **Command Help:**
   ```bash
   mdnotes --help
   mdnotes frontmatter --help
   mdnotes frontmatter ensure --help
   ```

2. **GitHub Issues:**
   Report bugs and feature requests at: https://github.com/eoinhurrell/mdnotes/issues

3. **Community:**
   Join discussions about Obsidian automation and mdnotes usage.

## Performance Reference

### Benchmark Results

Typical performance on modern hardware:

| Operation | 100 Files | 1000 Files | 10000 Files |
|-----------|-----------|------------|-------------|
| Frontmatter Ensure | 4ms | 46ms | 542ms |
| Type Validation | 11ms | 113ms | 1.3s |
| Heading Analysis | 8ms | 82ms | 890ms |
| Link Parsing | 6ms | 58ms | 630ms |

### Optimization Features

- **Parallel Processing:** Use `--workers N` for CPU-intensive operations
- **Memory Management:** Automatic chunking for large vaults
- **Streaming Mode:** Process files without loading all into memory
- **Smart Batching:** Optimize operation order for efficiency

### Best Practices

1. **Always use `--dry-run` first**
2. **Start with small test directories**
3. **Back up important vaults before bulk operations**
4. **Use configuration files for repeated operations**
5. **Monitor performance with `--verbose`**
6. **Leverage parallel processing for large vaults**