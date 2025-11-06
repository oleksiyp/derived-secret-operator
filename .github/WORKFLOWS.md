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

**Purpose**: Continuous validation + snapshot publishing

**Jobs**:
- ✅ Lint, Test, Security (same as PR)
- ✅ Helm Lint, Test & Publish Snapshot:
  - Creates snapshot versions: `0.1.0-dev.abc123`
  - Pushes to `oci://ghcr.io/oleksiyp/charts/derived-secret-operator:0.1.0-dev.abc123`
  - Doesn't modify Chart.yaml in git (version bump happens via release-please)
- ✅ Build & Push Edge Images:
  - `ghcr.io/oleksiyp/derived-secret-operator:edge`
  - `ghcr.io/oleksiyp/derived-secret-operator:main-<sha>`
  - Multi-arch: linux/amd64, linux/arm64
- ✅ E2E Tests

**Snapshots**: Every main push creates a snapshot Helm chart for testing latest changes.

### 3. Release Please Workflow (`release-please.yml`)
**Trigger**: Every push to `main` branch (but only publishes on Release PR merge)

**Purpose**: Automated releases based on Conventional Commits

**Phase 1 - Release PR Management** (automatic on every push):
- Scans git history for conventional commits
- Creates/updates a Release PR with:
  - Version bump (semver based on commit types)
  - Updated CHANGELOG.md
  - Updated Chart.yaml (version + appVersion)
- **Note**: No publishing happens yet - just PR creation/update

**Phase 2 - Release Publishing** (only when you manually merge the Release PR):
- ✅ Build & Push Versioned Docker Images:
  - `ghcr.io/oleksiyp/derived-secret-operator:v1.2.3`
  - `ghcr.io/oleksiyp/derived-secret-operator:latest`
  - Multi-arch: linux/amd64, linux/arm64
- ✅ Package & Push Official Helm Chart:
  - `oci://ghcr.io/oleksiyp/charts/derived-secret-operator:1.2.3`
- ✅ Generate & Upload installation YAML
- ✅ Create GitHub Release with:
  - Changelog from CHANGELOG.md
  - Installation instructions
  - Attached install.yaml

**Control**: You decide when to release by merging the Release PR. No automatic releases!

## Workflow Responsibilities

| Workflow | Validation | Snapshot Images | Snapshot Helm | Release Images | Release Helm |
|----------|-----------|----------------|---------------|----------------|--------------|
| PR Build | ✅ | ❌ | ❌ | ❌ | ❌ |
| CI | ✅ | ✅ (edge) | ✅ (dev) | ❌ | ❌ |
| Release-Please | ❌ | ❌ | ❌ | ✅ (on PR merge) | ✅ (on PR merge) |

## Why This Structure?

### Problem: Helm Chart Version Conflicts
If CI published official Helm chart versions on every push, it would try to push the same version (e.g., 0.1.0) multiple times to the OCI registry, which **rejects duplicate versions**.

### Solution: Snapshot vs Release Publishing
- **CI workflow**: Publishes snapshot Helm charts with unique versions (e.g., `0.1.0-dev.abc123`)
- **Release-please workflow**: Publishes official release versions (e.g., `0.1.0`) only when you merge Release PR

### Benefits
- ✅ No version conflicts in OCI registry
- ✅ Snapshots available immediately for testing
- ✅ Manual control over official releases
- ✅ Fast PR validation (amd64 only)
- ✅ Edge images + snapshot Helm charts for testing latest main
- ✅ Automated Release PRs with proper versioning
- ✅ All artifacts in sync (Docker + Helm + YAML)

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
