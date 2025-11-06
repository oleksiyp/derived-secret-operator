# GitHub Actions Workflows

This document explains the CI/CD workflow structure for this project.

## Workflow Overview

We use **2 streamlined workflows**:

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

### 2. CI/CD Workflow (`ci.yml`)
**Trigger**: Every push to `main` branch

**Purpose**: Complete pipeline - validation, snapshots, and releases

**Four phases in one workflow:**

#### Phase 1: Validation
- ✅ Lint, Test, Security, E2E (same as PR workflow)

#### Phase 2: Snapshot Publishing (always runs)
- ✅ Snapshot Docker Images:
  - `ghcr.io/oleksiyp/derived-secret-operator:edge`
  - `ghcr.io/oleksiyp/derived-secret-operator:main-<sha>`
  - Multi-arch: linux/amd64, linux/arm64
- ✅ Snapshot Helm Charts:
  - `oci://ghcr.io/oleksiyp/charts/derived-secret-operator:0.1.0-dev.abc123`
  - Unique version per commit (no conflicts)

#### Phase 3: Release-Please (always runs)
- Scans git history for conventional commits
- Creates/updates a Release PR automatically
- **Note**: No publishing yet - just PR management

#### Phase 4: Official Release Publishing (only when Release PR is merged)
- ✅ Release Docker Images:
  - `ghcr.io/oleksiyp/derived-secret-operator:v1.2.3`
  - `ghcr.io/oleksiyp/derived-secret-operator:latest`
  - Multi-arch, signed with cosign, includes SBOM
- ✅ Release Helm Chart:
  - `oci://ghcr.io/oleksiyp/charts/derived-secret-operator:1.2.3`
- ✅ Release Artifacts:
  - GitHub Release with changelog
  - Installation YAML attached
  - Installation instructions

**Control**: Merge the Release PR when you're ready to publish an official release!

## Workflow Responsibilities

| Workflow | Validation | Snapshots | Release PR | Official Release |
|----------|-----------|-----------|------------|------------------|
| PR Build | ✅ | ❌ | ❌ | ❌ |
| CI/CD | ✅ | ✅ (always) | ✅ (always) | ✅ (on PR merge) |

**One workflow to rule them all!** The CI/CD workflow handles everything.

## Why This Structure?

### Previous Problems (4 workflows!)
- ❌ Duplication between `ci.yml`, `release-please.yml`, and `release.yml`
- ❌ Confusing separation of concerns
- ❌ Hard to understand what runs when

### Solution: Single Consolidated CI/CD Workflow
- ✅ **One workflow** handles everything for main branch
- ✅ **Clear phases** within the workflow
- ✅ **Automatic snapshots** on every push (unique versions)
- ✅ **Automatic Release PRs** (no manual intervention needed)
- ✅ **Manual release control** (merge Release PR when ready)

### Benefits
- ✅ **Simple**: 2 workflows total (PR + CI/CD)
- ✅ **No duplication**: Single source of truth
- ✅ **No version conflicts**: Unique snapshot versions
- ✅ **Snapshots immediately available**: Test latest changes instantly
- ✅ **Manual release control**: You decide when to publish
- ✅ **Secure releases**: Signed images, SBOM, provenance
- ✅ **All artifacts in sync**: Docker + Helm + YAML

## Artifact Versioning Strategy

### Snapshot Artifacts (CI Workflow - Every Push)

**Docker Images**:
```bash
# Always latest main branch
ghcr.io/oleksiyp/derived-secret-operator:edge

# Specific commit (for debugging)
ghcr.io/oleksiyp/derived-secret-operator:main-abc123
```

**Helm Charts**:
```bash
# Snapshot version with commit hash
oci://ghcr.io/oleksiyp/charts/derived-secret-operator:0.1.0-dev.abc123

# Install snapshot
helm install my-app \
  oci://ghcr.io/oleksiyp/charts/derived-secret-operator \
  --version 0.1.0-dev.abc123
```

### Release Artifacts (Release-Please Workflow - Manual)

**Docker Images**:
```bash
# Specific version (immutable)
ghcr.io/oleksiyp/derived-secret-operator:v1.2.3

# Latest stable release
ghcr.io/oleksiyp/derived-secret-operator:latest
```

**Helm Charts**:
```bash
# Official release version
oci://ghcr.io/oleksiyp/charts/derived-secret-operator:1.2.3

# Install official release
helm install my-app \
  oci://ghcr.io/oleksiyp/charts/derived-secret-operator \
  --version 1.2.3
```

Charts are versioned according to Chart.yaml, which is updated by release-please when you merge the Release PR.

## Making a Release

### Daily Development (Automatic)
1. **Commit with conventional format** and push to main:
   ```bash
   git commit -m "feat: add new feature"
   git push origin main
   ```

2. **Automatic snapshots published**:
   - Docker: `ghcr.io/oleksiyp/derived-secret-operator:edge`
   - Helm: `oci://ghcr.io/oleksiyp/charts/derived-secret-operator:0.1.0-dev.abc123`

3. **Release-please creates/updates Release PR** (automatic, but not merged)

### Creating an Official Release (Manual)
1. **Review the Release PR** created by release-please:
   - Check version bump is correct
   - Review CHANGELOG.md changes
   - Verify Chart.yaml updates

2. **Merge the Release PR** (this is the manual trigger!)

3. **Automated publishing happens**:
   - Docker images: `v1.2.3` + `latest`
   - Helm chart: `oci://ghcr.io/oleksiyp/charts/derived-secret-operator:1.2.3`
   - GitHub Release with install.yaml
   - Installation instructions

**Key Point**: You control WHEN releases happen by choosing when to merge the Release PR. Snapshots are always available for testing.

See [README.md Release Process section](../README.md#release-process) for detailed commit message format.

## Troubleshooting

### "How do I test latest main changes?"
**Use snapshot artifacts**:
```bash
# Docker
docker pull ghcr.io/oleksiyp/derived-secret-operator:edge

# Helm (get version from CI logs)
helm install my-app \
  oci://ghcr.io/oleksiyp/charts/derived-secret-operator \
  --version 0.1.0-dev.abc123
```

### "Snapshot Helm charts not publishing"
**Cause**: CI workflow may have failed
**Solution**: Check CI workflow run in GitHub Actions, look for Helm job

### "Release-please PR not being created"
**Cause**: No conventional commits since last release
**Solution**: Ensure commits follow conventional format (feat:, fix:, etc.)

### "Official release didn't publish"
**Cause**: publish-release job only runs when Release PR is merged
**Solution**: Merge the Release PR created by release-please
