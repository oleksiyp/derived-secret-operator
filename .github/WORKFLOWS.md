# GitHub Actions Workflows

Simple 3-workflow CI/CD setup for PR-based development.

## Overview

```
┌─────────────┐
│   PR Build  │ ← Validation only (fast)
└─────────────┘

┌─────────────┐
│     CI      │ ← Validation + Snapshots (after merge)
└─────────────┘

┌─────────────┐
│  Release    │ ← Manual trigger for official releases
└─────────────┘
```

## Workflows

### 1. PR Build (`pr.yml`)

**Trigger:** Pull requests to `main`

**Purpose:** Fast validation before merge

**Jobs:**
- Lint (golangci-lint)
- Test (unit + coverage)
- Security (gosec)
- Helm validation
- E2E tests
- Build (amd64 only, no push)

**Features:**
- Auto-cancels outdated builds
- Runs in ~5 minutes
- Must pass before merge

### 2. CI (`ci.yml`)

**Trigger:** Push to `main` (after PR merge)

**Purpose:** Validation + snapshot publishing

**Jobs:**
- Validate (same as PR)
- E2E tests
- Publish snapshot Docker images:
  - `ghcr.io/oleksiyp/derived-secret-operator:edge`
  - `ghcr.io/oleksiyp/derived-secret-operator:sha-abc123`
- Publish snapshot Helm chart:
  - `oci://ghcr.io/oleksiyp/charts/derived-secret-operator:0.1.0-dev.abc123`

**Snapshots:** Updated on every main push for immediate testing

### 3. Release (`release.yml`)

**Trigger:** Manual (workflow_dispatch)

**Purpose:** Create official versioned releases

**How to release:**
1. Go to GitHub Actions
2. Select "Release" workflow
3. Click "Run workflow"
4. Enter version (e.g., `1.0.0`)
5. Click "Run workflow" button

**What happens:**
- Validates version format (semver)
- Checks tag doesn't exist
- Updates Chart.yaml versions
- Builds multi-arch Docker images (amd64 + arm64)
- Signs images with cosign
- Generates SBOM
- Packages Helm chart
- Creates git tag `v1.0.0`
- Creates GitHub Release with:
  - Release notes
  - install.yaml
  - SBOM
  - Installation instructions

**Published artifacts:**
- Docker: `ghcr.io/oleksiyp/derived-secret-operator:v1.0.0` + `latest`
- Helm: `oci://ghcr.io/oleksiyp/charts/derived-secret-operator:1.0.0`

## Workflow Comparison

| Feature | PR Build | CI | Release |
|---------|----------|-----|---------|
| Trigger | PR | Push to main | Manual |
| Validation | ✅ | ✅ | ❌ |
| Snapshots | ❌ | ✅ | ❌ |
| Versioned Release | ❌ | ❌ | ✅ |
| Multi-arch | ❌ (amd64) | ✅ | ✅ |
| Sign Images | ❌ | ❌ | ✅ |
| SBOM | ❌ | ❌ | ✅ |

## Development Flow

### Daily Development

```bash
# 1. Create feature branch
git checkout -b feature/my-feature

# 2. Make changes and commit
git commit -m "Add new feature"
git push origin feature/my-feature

# 3. Create PR
# → PR Build runs (validation)

# 4. Merge PR
# → CI runs (validation + snapshots)
```

### Testing Snapshots

```bash
# Use edge tag (always latest main)
docker pull ghcr.io/oleksiyp/derived-secret-operator:edge

# Or specific commit
docker pull ghcr.io/oleksiyp/derived-secret-operator:sha-abc123

# Helm (get version from CI logs)
helm install my-app \
  oci://ghcr.io/oleksiyp/charts/derived-secret-operator \
  --version 0.1.0-dev.abc123
```

### Creating a Release

1. **Decide on version** (semver):
   - Major (1.0.0 → 2.0.0): Breaking changes
   - Minor (1.0.0 → 1.1.0): New features
   - Patch (1.0.0 → 1.0.1): Bug fixes

2. **Trigger release**:
   - Go to: https://github.com/oleksiyp/derived-secret-operator/actions/workflows/release.yml
   - Click "Run workflow"
   - Enter version
   - Click "Run workflow"

3. **Release publishes**:
   - Docker images with version tags
   - Helm chart with version
   - GitHub Release with changelog
   - All signed with cosign + SBOM

## Versioning Strategy

### Snapshots (Automatic)
```
Docker:  edge, sha-abc123
Helm:    0.1.0-dev.abc123
When:    Every main push
Purpose: Testing latest changes
```

### Releases (Manual)
```
Docker:  v1.0.0, latest
Helm:    1.0.0
When:    Manual workflow trigger
Purpose: Official stable releases
```

## Why This Structure?

### ✅ Advantages
- **Simple**: Easy to understand PR → CI → Release flow
- **No automatic releases**: Full control over when to release
- **Snapshots for testing**: Latest main always available
- **Clean versioning**: Semver for releases, dev versions for snapshots
- **Security**: Signed releases with SBOM
- **No commit message requirements**: Write natural commit messages

### ❌ What We Don't Use
- **release-please**: Too complex for PR-based workflow
- **Automatic releases**: You control when to release
- **Conventional commits**: Not needed for this workflow

## Troubleshooting

### PR Build Failing
**Check:** GitHub Actions → PR checks
**Common causes:** Lint errors, test failures, E2E issues

### Snapshots Not Publishing
**Check:** CI workflow logs
**Verify:** GHCR permissions, workflow completed

### Release Failing
**Common causes:**
- Invalid version format (must be semver: 1.0.0)
- Tag already exists
- GHCR permission issues

**Fix:**
- Use correct semver format
- Delete existing tag if needed
- Check GHCR token permissions

## Best Practices

### Commit Messages
Write clear, descriptive messages:
```bash
git commit -m "Add support for custom parameters"
git commit -m "Fix memory leak in controller"
git commit -m "Update Helm chart defaults"
```

### Before Release
- [ ] All PRs merged
- [ ] Snapshots tested
- [ ] CHANGELOG updated (if you maintain one)
- [ ] Breaking changes documented
- [ ] Version number decided

### After Release
- [ ] Test official release artifacts
- [ ] Update documentation if needed
- [ ] Announce release (if applicable)

## Resources

- [Kubebuilder](https://book.kubebuilder.io/)
- [Semantic Versioning](https://semver.org/)
- [GitHub Actions](https://docs.github.com/en/actions)
