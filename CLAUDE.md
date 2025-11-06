# Derived Secret Operator - Development Guide

Guide for Claude Code when working on this project.

## Project Overview

Kubernetes operator that derives secrets deterministically from a master password using Argon2id KDF.

**Tech Stack:**
- Go 1.24+
- Kubebuilder v4
- Kubernetes 1.31+
- Multi-arch: linux/amd64, linux/arm64

## Development Workflow

### 1. Work in Branches & PRs
```bash
git checkout -b feature/my-feature
# make changes
git commit -m "Add new feature"
git push origin feature/my-feature
# Create PR on GitHub
```

### 2. PR Gets Validated
- Lint, test, security scan
- E2E tests
- Helm chart validation
- Must pass before merge

### 3. Merge PR → Snapshots Published
- Docker: `ghcr.io/oleksiyp/derived-secret-operator:edge`
- Docker: `ghcr.io/oleksiyp/derived-secret-operator:sha-abc123`
- Helm: `oci://ghcr.io/oleksiyp/charts/derived-secret-operator:0.1.0-dev.abc123`

### 4. Release When Ready
- Go to GitHub Actions → Release workflow
- Click "Run workflow"
- Enter version (e.g., `1.0.0`)
- Official release published automatically

## Commit Messages

Simple, descriptive messages. No special format required.

**Good examples:**
```bash
git commit -m "Add support for custom Argon2id parameters"
git commit -m "Fix nil pointer in DerivedSecret reconciliation"
git commit -m "Update Helm chart memory limits"
git commit -m "Add examples to README"
```

**Always include co-authorship:**
```
Add support for custom Argon2id parameters

🤖 Generated with [Claude Code](https://claude.com/claude-code)
via [Happy](https://happy.engineering)

Co-Authored-By: Claude <noreply@anthropic.com>
Co-Authored-By: Happy <yesreply@happy.engineering>
```

## Workflows

### PR Workflow (`pr.yml`)
**Triggers:** Pull requests to main
**Purpose:** Fast validation
**Jobs:**
- Lint (golangci-lint)
- Test (unit tests with coverage)
- Security scan (gosec)
- Helm validation
- E2E tests
- Build test (amd64 only, no push)

**Auto-cancels** outdated builds when new commits pushed.

### CI Workflow (`ci.yml`)
**Triggers:** Push to main (after PR merge)
**Purpose:** Validation + snapshot publishing
**Jobs:**
- Validate (lint, test, security, E2E)
- Snapshot Docker images → `edge` + `sha-xxx`
- Snapshot Helm chart → `0.1.0-dev.xxx`

**Snapshots** are published on every main push for testing.

### Release Workflow (`release.yml`)
**Triggers:** Manual (workflow_dispatch)
**Purpose:** Create official versioned releases
**Steps:**
1. Go to GitHub Actions
2. Select "Release" workflow
3. Click "Run workflow"
4. Enter version (e.g., `1.0.0`)
5. Release publishes automatically:
   - Docker images: `v1.0.0` + `latest`
   - Helm chart: `1.0.0`
   - GitHub Release with install.yaml
   - Signed with cosign + SBOM

## Testing Snapshots

```bash
# Docker
docker pull ghcr.io/oleksiyp/derived-secret-operator:edge

# Helm (check CI logs for exact version)
helm install my-app \
  oci://ghcr.io/oleksiyp/charts/derived-secret-operator \
  --version 0.1.0-dev.abc123
```

## Testing Official Releases

```bash
# Docker
docker pull ghcr.io/oleksiyp/derived-secret-operator:v1.0.0

# Helm
helm install my-app \
  oci://ghcr.io/oleksiyp/charts/derived-secret-operator \
  --version 1.0.0
```

## Common Tasks

### Update Dependencies
```bash
go get -u ./...
go mod tidy
git commit -m "Update Go dependencies"
```

### Regenerate CRDs
```bash
make manifests
git commit -m "Regenerate CRDs"
```

### Sync Kustomize and Helm

**CRITICAL:** Kustomize manifests and Helm charts must be kept in sync.

When changing RBAC, CRDs, or deployment configs:
1. Update `config/` directory (kustomize source)
2. Mirror changes to `charts/derived-secret-operator/`
3. Test both installation methods

**RBAC Mapping:**
- `config/rbac/role.yaml` → `charts/.../templates/clusterrole.yaml`
- `config/rbac/leader_election_role.yaml` → `charts/.../templates/leader-election-role.yaml`
- `config/rbac/role_binding.yaml` → `charts/.../templates/clusterrolebinding.yaml`
- `config/rbac/leader_election_role_binding.yaml` → `charts/.../templates/leader-election-rolebinding.yaml`

**CRD Mapping:**
- `config/crd/bases/*.yaml` → `charts/.../crds/*.yaml`

**Deployment Config:**
- `config/manager/manager.yaml` → `charts/.../templates/deployment.yaml` + `values.yaml`

**Example workflow:**
```bash
# 1. Update kustomize manifests
vim config/rbac/role.yaml

# 2. Update corresponding Helm template
vim charts/derived-secret-operator/templates/clusterrole.yaml

# 3. Bump Helm chart version (patch)
vim charts/derived-secret-operator/Chart.yaml

# 4. Test both installation methods
make build-installer IMG=test:latest
kubectl apply -f dist/install.yaml

helm install test charts/derived-secret-operator --dry-run

# 5. Commit with descriptive message
git commit -m "Add RBAC permission for X resource"
```

### Update Helm Chart
```bash
# Modify charts/derived-secret-operator/*
# Version is updated automatically during release
git commit -m "Add new Helm configuration option"
```

### Lint Fixes
```bash
make lint-fix
git commit -m "Fix linting issues"
```

## Code Guidelines

### Controller Best Practices
- Use structured logging with controller-runtime logger
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
- Memory limits: 256Mi

## Versioning

Use semantic versioning (semver):
- **Major** (1.0.0 → 2.0.0): Breaking changes
- **Minor** (1.0.0 → 1.1.0): New features, backwards compatible
- **Patch** (1.0.0 → 1.0.1): Bug fixes, backwards compatible

## Important Files

- `charts/derived-secret-operator/` - Helm chart
- `config/manager/manager.yaml` - Operator deployment config
- `.golangci.yml` - Linter configuration
- `.github/workflows/` - CI/CD workflows

## Troubleshooting

### CI Failing
- Check GitHub Actions logs
- Common issues: lint errors, test failures, OOM

### Snapshot Not Publishing
- Check CI workflow completed successfully
- Verify GHCR permissions

### Release Failing
- Check version format (must be semver: 1.0.0)
- Verify tag doesn't already exist
- Check GHCR permissions

## Resources

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Workflow Documentation](.github/WORKFLOWS.md)
- [Project README](README.md)
