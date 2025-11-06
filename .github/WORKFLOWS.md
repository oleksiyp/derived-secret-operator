# GitHub Actions Workflows

This document explains the CI/CD workflow structure for this project.

## Workflow Overview

We use three main workflows with clear separation of concerns:

### 1. PR Build Workflow (`pr.yml`)
**Trigger**: Pull requests to `main` branch

**Purpose**: Fast validation of PRs before merge

**Jobs**:
- ✅ Lint (Go code with golangci-lint)
- ✅ Test (unit tests with coverage)
- ✅ Security (Gosec scanner)
- ✅ Helm (lint & template validation)
- ✅ E2E Tests (Kind cluster integration tests)
- ✅ Build Image (amd64 only for speed, no push)

**Features**:
- Auto-cancels outdated builds when new commits pushed
- Runs in ~5-10 minutes (amd64 only)

### 2. CI Workflow (`ci.yml`)
**Trigger**: Every push to `main` branch

**Purpose**: Continuous validation + edge image publishing

**Jobs**:
- ✅ Lint, Test, Security (same as PR)
- ✅ Helm Lint & Test (validates but does NOT publish)
- ✅ Build & Push Edge Images:
  - `ghcr.io/oleksiyp/derived-secret-operator:edge`
  - `ghcr.io/oleksiyp/derived-secret-operator:main-<sha>`
  - Multi-arch: linux/amd64, linux/arm64
- ✅ E2E Tests

**Important**: This workflow does NOT publish Helm charts to avoid version conflicts.

### 3. Release Please Workflow (`release-please.yml`)
**Trigger**: Every push to `main` branch

**Purpose**: Automated releases based on Conventional Commits

**Phase 1 - Release PR Management**:
- Scans git history for conventional commits
- Creates/updates a Release PR with:
  - Version bump (semver based on commit types)
  - Updated CHANGELOG.md
  - Updated Chart.yaml (version + appVersion)

**Phase 2 - Release Publishing** (when Release PR is merged):
- ✅ Build & Push Versioned Docker Images:
  - `ghcr.io/oleksiyp/derived-secret-operator:v1.2.3`
  - `ghcr.io/oleksiyp/derived-secret-operator:latest`
  - Multi-arch: linux/amd64, linux/arm64
- ✅ Package & Push Helm Chart:
  - `oci://ghcr.io/oleksiyp/charts/derived-secret-operator:1.2.3`
- ✅ Generate & Upload installation YAML
- ✅ Create GitHub Release with:
  - Changelog from CHANGELOG.md
  - Installation instructions
  - Attached install.yaml

## Workflow Responsibilities

| Workflow | Validation | Edge Images | Versioned Images | Helm Charts |
|----------|-----------|-------------|------------------|-------------|
| PR Build | ✅ | ❌ | ❌ | ❌ (validate only) |
| CI | ✅ | ✅ | ❌ | ❌ (validate only) |
| Release-Please | ❌ | ❌ | ✅ | ✅ (publish) |

## Why This Structure?

### Problem: Helm Chart Version Conflicts
If CI published Helm charts on every push, it would try to push the same version (e.g., 0.1.0) multiple times to the OCI registry, which **rejects duplicate versions**.

### Solution: Single Source of Publishing
- **CI workflow**: Only validates Helm charts, never publishes them
- **Release-please workflow**: Exclusive publisher of versioned artifacts

### Benefits
- ✅ No version conflicts in OCI registry
- ✅ Clear separation of concerns
- ✅ Fast PR validation (amd64 only)
- ✅ Edge images for testing latest main
- ✅ Automated releases with proper versioning
- ✅ All artifacts in sync (Docker + Helm + YAML)

## Image Tagging Strategy

### Edge Images (CI Workflow)
```bash
# Always latest main branch
ghcr.io/oleksiyp/derived-secret-operator:edge

# Specific commit (for debugging)
ghcr.io/oleksiyp/derived-secret-operator:main-abc123def
```

### Release Images (Release-Please Workflow)
```bash
# Specific version (immutable)
ghcr.io/oleksiyp/derived-secret-operator:v1.2.3

# Latest stable release
ghcr.io/oleksiyp/derived-secret-operator:latest
```

## Helm Chart Publishing

Only the **release-please workflow** publishes Helm charts:

```bash
# Published when Release PR is merged
oci://ghcr.io/oleksiyp/charts/derived-secret-operator:1.2.3
```

Charts are versioned according to Chart.yaml, which is updated by release-please.

## Making a Release

1. **Commit with conventional format**:
   ```bash
   git commit -m "feat: add new feature"  # Minor bump
   git commit -m "fix: resolve bug"       # Patch bump
   git commit -m "feat!: breaking change" # Major bump
   ```

2. **Push to main** (or merge PR with squash-merge)

3. **Release-please creates/updates PR** automatically

4. **Review and merge Release PR**

5. **Automated publishing**:
   - Docker images tagged with version
   - Helm chart published to GHCR
   - GitHub Release created with assets
   - Installation instructions added

See [README.md Release Process section](../README.md#release-process) for detailed commit message format.

## Troubleshooting

### "OCI registry rejects duplicate version"
**Cause**: Trying to push same Helm chart version twice
**Solution**: Version must be bumped in Chart.yaml via release-please

### "Edge images not updating"
**Cause**: CI workflow may have failed
**Solution**: Check CI workflow run in GitHub Actions

### "Release didn't create Helm chart"
**Cause**: Chart.yaml version wasn't updated
**Solution**: Ensure release-please updated Chart.yaml in Release PR
