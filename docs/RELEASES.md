# Release Process Documentation

This document describes the complete release process for mdnotes, from preparation to post-release verification.

## Overview

The mdnotes project uses an automated CI/CD pipeline built with GitHub Actions and GoReleaser to create comprehensive releases with multiple distribution channels.

## Release Components

Each release includes:

### Binaries
- **Linux**: AMD64 and ARM64 architectures
- **macOS**: Intel (AMD64) and Apple Silicon (ARM64)
- **Windows**: AMD64 architecture
- All binaries are statically compiled with optimized builds

### Distribution Formats
- **Archives**: tar.gz for Unix systems, zip for Windows
- **Container Images**: Multi-architecture Docker images
- **Package Managers**: Homebrew formula, Linux packages (deb, rpm, apk)
- **SBOM**: Software Bill of Materials for supply chain security

### Assets
- **Shell Completions**: Bash, Zsh, Fish, and PowerShell
- **Documentation**: Complete documentation bundle
- **Checksums**: SHA256 checksums for all binaries
- **Security**: SBOM and signature verification

## Release Types

### Regular Releases (v1.2.3)
- Full validation and testing
- All distribution channels updated
- Public release with announcements

### Pre-releases (v1.2.3-alpha, v1.2.3-beta, v1.2.3-rc)
- Same build process as regular releases
- Marked as pre-release on GitHub
- Limited distribution (no package manager updates)

## Release Process

### 1. Preparation Phase

#### Prerequisites
- Clean working directory on `main` branch
- All tests passing locally
- Documentation updated
- CHANGELOG.md updated (if maintained)

#### Pre-release Validation
```bash
# Run the preparation script
./scripts/prepare-release.sh v1.2.3

# This script will:
# - Validate version format
# - Check branch and working directory
# - Run comprehensive tests
# - Validate cross-platform builds
# - Check security vulnerabilities
# - Test completions functionality
```

### 2. Release Creation

#### Automated Release (Recommended)
```bash
# Create and push the tag
git tag v1.2.3
git push origin v1.2.3

# This triggers the GitHub Actions release workflow
```

#### Manual Release (if needed)
```bash
# Use GitHub CLI
gh release create v1.2.3 --title "Release v1.2.3" --notes-file release-notes.md

# Or use the GitHub web interface
```

### 3. Automated Pipeline

The GitHub Actions release workflow performs:

#### Validation Stage
- Tag format validation
- Version extraction and pre-release detection

#### Testing Stage
- Comprehensive test suite with race detection
- Benchmark execution
- Completion functionality testing

#### Security Stage
- Security vulnerability scanning (gosec, govulncheck)
- Dependency security review

#### Build Verification Stage
- Cross-platform build testing
- Binary functionality verification

#### Release Stage
- GoReleaser execution
- Docker image building and publishing
- Asset generation and upload
- SBOM generation

#### Post-Release Stage
- Asset verification
- Docker image testing
- Release artifact validation

### 4. Post-Release Verification

```bash
# Run the verification script
./scripts/verify-release.sh v1.2.3

# This script will:
# - Verify all expected assets exist
# - Test binary downloads and functionality
# - Validate Docker images
# - Check checksums
# - Verify SBOM format
```

## Quality Gates

The release process includes multiple quality gates:

### Pre-Release Gates
- ✅ All tests pass (unit, integration, benchmarks)
- ✅ Security scans pass (gosec, govulncheck)
- ✅ Cross-platform builds succeed
- ✅ Completion functionality works
- ✅ Docker build succeeds

### Release Gates
- ✅ GoReleaser validation passes
- ✅ All target platforms build successfully
- ✅ Container images build and publish
- ✅ SBOM generation succeeds

### Post-Release Gates
- ✅ All expected assets are present
- ✅ Binary functionality verified
- ✅ Docker images work correctly
- ✅ Checksums validate

## Rollback Procedures

### Failed Release
If a release fails during the pipeline:

1. **Check the workflow logs** for specific errors
2. **Fix the underlying issue**
3. **Delete the tag** if it was created: `git tag -d v1.2.3 && git push origin :v1.2.3`
4. **Re-run the preparation script** to ensure fixes work
5. **Create a new tag** with the same version

### Bad Release
If a release is published but has issues:

1. **Create a new patch version** (e.g., v1.2.4) with fixes
2. **Mark the bad release as pre-release** on GitHub to hide it
3. **Follow normal release process** for the fix
4. **Document the issue** in release notes

## Distribution Channels

### GitHub Releases
- Primary distribution channel
- All assets available immediately after release
- Automatic changelog generation

### Container Registry (GHCR)
- `ghcr.io/eoinhurrell/mdnotes:v1.2.3` (versioned)
- `ghcr.io/eoinhurrell/mdnotes:latest` (latest stable)
- Multi-architecture support (AMD64, ARM64)

### Homebrew (if configured)
- Personal tap: `eoinhurrell/tap/mdnotes`
- Automatic formula updates via GoReleaser
- macOS and Linux support

### Linux Packages (if configured)
- Debian/Ubuntu: `.deb` packages
- RedHat/CentOS: `.rpm` packages
- Alpine: `.apk` packages
- Arch Linux: packages for AUR

## Security Considerations

### Supply Chain Security
- All builds use reproducible builds where possible
- SBOM generation for dependency tracking
- Container image vulnerability scanning
- Dependency security monitoring

### Release Signing (Future Enhancement)
- Binary signing with Sigstore
- Container image signing
- Release attestation

## Monitoring and Metrics

### Release Success Metrics
- Build time and success rate
- Download statistics
- Container image pulls
- Security scan results

### Post-Release Monitoring
- Error reports from new version
- Performance regression detection
- User feedback monitoring

## Troubleshooting

### Common Issues

#### GoReleaser Fails
- Check `.goreleaser.yaml` syntax
- Verify all required environment variables
- Ensure Docker daemon is running
- Check GitHub token permissions

#### Cross-Platform Build Fails
- Verify Go version compatibility
- Check for platform-specific code issues
- Ensure CGO is properly disabled

#### Security Scan Failures
- Review and address security vulnerabilities
- Update dependencies if needed
- Consider adding security exceptions if false positives

#### Container Build Fails
- Check Dockerfile syntax
- Verify base image availability
- Ensure build context is correct

### Debug Commands

```bash
# Test GoReleaser locally
goreleaser check
goreleaser release --snapshot --clean

# Test Docker build
docker build -t mdnotes:test .

# Test cross-platform builds
GOOS=linux GOARCH=amd64 go build ./cmd
GOOS=darwin GOARCH=arm64 go build ./cmd
GOOS=windows GOARCH=amd64 go build ./cmd

# Test completions
go build -o mdnotes ./cmd
./mdnotes completion bash | head -10
./mdnotes __complete frontmatter ensure --field ""
```

## Release Schedule

### Regular Schedule
- **Patch releases**: As needed for bug fixes
- **Minor releases**: Monthly or bi-monthly for features
- **Major releases**: Quarterly or when breaking changes accumulate

### Emergency Releases
- Critical security vulnerabilities
- Major functionality breaking bugs
- Data corruption issues

## Communication

### Internal
- Release preparation checklist
- Code review for release-related changes
- Testing coordination

### External (if public project)
- Release announcements
- Breaking change notifications
- Migration guides for major versions

## Continuous Improvement

### Metrics Collection
- Release frequency and success rate
- Time from tag to asset availability
- User adoption of new versions

### Process Refinement
- Regular review of release process
- Automation of manual steps
- Quality gate optimization

This document should be updated as the release process evolves.