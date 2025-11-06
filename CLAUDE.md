# Derived Secret Operator - Development Guide

This document provides guidance for Claude Code when working on this project.

## Project Overview

Kubernetes operator that derives secrets deterministically from a master password using Argon2id KDF.

**Tech Stack:**
- Go 1.24+
- Kubebuilder v4
- controller-runtime v0.22.1
- Kubernetes 1.31+

## Commit Message Format

**CRITICAL**: This project uses [Conventional Commits](https://www.conventionalcommits.org/) with [release-please](https://github.com/googleapis/release-please) for automated releases.

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types and Version Bumps

| Type | Version Bump | Example |
|------|--------------|---------|
| `fix:` | Patch (0.1.0 → 0.1.1) | Bug fixes |
| `feat:` | Minor (0.1.0 → 0.2.0) | New features |
| `feat!:` or `BREAKING CHANGE:` | Major (0.1.0 → 1.0.0) | Breaking changes |
| `docs:` | None | Documentation only |
| `style:` | None | Formatting, whitespace |
| `refactor:` | None | Code restructuring |
| `perf:` | Patch | Performance improvements |
| `test:` | None | Adding tests |
| `build:` | Patch | Build system changes |
| `ci:` | None | CI configuration |
| `chore:` | None | Maintenance tasks |

### Commit Message Examples

```bash
# Feature (minor version bump)
feat: add support for custom Argon2id parameters

Allow users to specify time, memory, and parallelism parameters
for Argon2id key derivation in MasterPassword CRD.

# Bug fix (patch version bump)
fix: resolve nil pointer in DerivedSecret reconciliation

Added nil check before accessing master password reference.

Fixes #123

# Breaking change (major version bump)
feat!: change DerivedSecret CRD structure

BREAKING CHANGE: The spec.keys field is now an array instead of a map.
Users must update their DerivedSecret resources.

Migration guide available in docs/migration.md

# Documentation (no version bump)
docs: add examples for complex secret derivation patterns

# CI changes (no version bump)
ci: optimize Docker build cache configuration
```

### Scope Examples

Common scopes for this project:
- `api`: CRD changes
- `controller`: Controller logic
- `crypto`: Cryptography/KDF code
- `helm`: Helm chart changes
- `ci`: CI/CD workflows
- `deps`: Dependency updates

### Footer Keywords

- `Fixes #123` - Links to issue
- `Closes #123` - Closes issue
- `BREAKING CHANGE: description` - Marks breaking changes
- `Release-As: 1.0.0` - Manually override version

## Commit Signature

**IMPORTANT**: Always include Happy and Claude co-authorship in commits:

```
🤖 Generated with [Claude Code](https://claude.com/claude-code)
via [Happy](https://happy.engineering)

Co-Authored-By: Claude <noreply@anthropic.com>
Co-Authored-By: Happy <yesreply@happy.engineering>
```

## Release Workflow

### Before Pushing to Main

**ALWAYS ASK** before pushing:
1. "Should I trigger a release by merging the Release PR?"
2. Wait for user confirmation

### Release Process

1. **Daily development** (automatic):
   ```bash
   git commit -m "feat: new feature"
   git push origin main
   ```
   → Snapshots published immediately
   → Release PR created/updated automatically

2. **Creating official release** (manual):
   - Review the Release PR created by release-please
   - Check version bump is appropriate
   - **Ask user**: "The Release PR is ready. Should I merge it to trigger the official release?"
   - If yes: Merge the Release PR
   - Result: Official v1.2.3 release published with Docker images, Helm chart, and GitHub Release

### Snapshot Testing

Snapshots are published on every main push:
```bash
# Docker
ghcr.io/oleksiyp/derived-secret-operator:edge

# Helm (check CI logs for exact version)
oci://ghcr.io/oleksiyp/charts/derived-secret-operator:0.1.0-dev.abc123
```

## CI/CD Workflows

### PR Workflow (`pr.yml`)
- Triggers: Pull requests to main
- Fast validation (amd64 only)
- No publishing

### CI/CD Workflow (`ci.yml`)
- Triggers: Every push to main
- **Phase 1**: Validation (lint, test, security, E2E)
- **Phase 2**: Publish snapshots (always)
- **Phase 3**: Create/update Release PR (always)
- **Phase 4**: Publish official release (only when Release PR is merged)

## Code Guidelines

### Controller Best Practices
- Always use structured logging with controller-runtime logger
- Return `ctrl.Result{RequeueAfter: duration}` for transient errors
- Use finalizers for cleanup logic
- Validate CRDs with kubebuilder markers

### Testing
- Unit tests: `go test ./...`
- E2E tests: `make test-e2e`
- Coverage target: >80%

### Security
- Operator runs as non-root (UID 65532)
- Read-only root filesystem
- No privilege escalation
- Secrets stored in etcd, encrypted at rest

## Common Tasks

### Update Dependencies
```bash
go get -u ./...
go mod tidy
git commit -m "chore(deps): update Go dependencies"
```

### Regenerate CRDs
```bash
make manifests
git commit -m "chore: regenerate CRDs"
```

### Update Helm Chart
```bash
# Modify charts/derived-secret-operator/*
# Let release-please bump version
git commit -m "feat(helm): add new configuration option"
```

### Lint Fixes
```bash
make lint-fix
git commit -m "style: fix linting issues"
```

## Branch Protection

- **main** branch is protected
- All changes must go through PRs
- CI must pass before merge
- Squash-merge preferred for clean history

## Important Files

- `.release-please-config.json` - Release configuration
- `.release-please-manifest.json` - Current version tracking
- `CHANGELOG.md` - Auto-generated by release-please
- `.golangci.yml` - Linter configuration
- `charts/derived-secret-operator/` - Helm chart

## Questions Before Action

Before performing these actions, **ALWAYS ASK THE USER**:

1. **Before pushing to main**:
   - "Ready to push? This will trigger CI/CD and create/update the Release PR."

2. **When Release PR exists**:
   - "I see there's a Release PR (#X) for version X.Y.Z. Should I merge it to trigger the official release?"

3. **For breaking changes**:
   - "This change might be breaking. Should I use `feat!:` or add `BREAKING CHANGE:` to the commit message?"

4. **For version overrides**:
   - "Should I include `Release-As: X.Y.Z` to manually set the version?"

## Troubleshooting

### Release PR Not Created
- Check conventional commit format
- Must have `feat:` or `fix:` since last release

### CI Failing
- Check GitHub Actions logs
- Common issues: lint errors, test failures, memory limits

### Helm Chart Conflicts
- Snapshots use unique versions (0.1.0-dev.abc123)
- Official releases use semver (0.1.0)
- No conflicts possible

## Resources

- [Conventional Commits](https://www.conventionalcommits.org/)
- [release-please](https://github.com/googleapis/release-please)
- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Workflow Documentation](.github/WORKFLOWS.md)
