# Release v{VERSION}

## Overview
Brief description of what this release includes.

## ✨ New Features
- Feature 1: Description of new feature
- Feature 2: Description of another feature

## 🐛 Bug Fixes
- Fix 1: Description of bug fix
- Fix 2: Description of another fix

## 🔧 Improvements
- Improvement 1: Description of enhancement
- Improvement 2: Description of optimization

## ⚠️ Breaking Changes
- Breaking change 1: Description and migration steps
- Breaking change 2: Description and migration steps

## 📦 Installation

### Download Binary
Download the appropriate binary for your platform from the assets below.

### Homebrew (macOS/Linux)
```bash
brew install eoinhurrell/tap/mdnotes
```

### Go Install
```bash
go install github.com/eoinhurrell/mdnotes/cmd@{TAG}
```

### Docker
```bash
docker run --rm -v $(pwd):/vault ghcr.io/eoinhurrell/mdnotes:{TAG} --help
```

## 🏗️ Shell Completions

Each archive includes shell completion scripts:
- Extract the archive
- Copy the appropriate completion script from `dist/completions/`
- Install according to your shell's documentation

## 🔒 Security & Verification

### Checksums
Verify downloads using the provided checksums:
```bash
sha256sum -c checksums.txt
```

### SBOM
This release includes a Software Bill of Materials (SBOM) for supply chain security.

## 📋 Full Changelog
**Full Changelog**: https://github.com/eoinhurrell/mdnotes/compare/{PREVIOUS_TAG}...{TAG}

## 🙏 Contributors
Thanks to all contributors who helped with this release!