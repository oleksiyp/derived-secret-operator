# Derived Secret Operator

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/oleksiyp/derived-secret-operator)](https://goreportcard.com/report/github.com/oleksiyp/derived-secret-operator)
[![GitHub release](https://img.shields.io/github/release/oleksiyp/derived-secret-operator.svg)](https://github.com/oleksiyp/derived-secret-operator/releases)

A Kubernetes operator that derives secrets deterministically from a master password using Argon2id KDF (Key Derivation Function).

## Features

- 🔐 **Cryptographically secure**: Uses Argon2id, a memory-hard KDF resistant to GPU attacks
- 🔄 **Deterministic**: Same input always produces the same output, enabling GitOps workflows
- 🎯 **Flexible**: Generate secrets of any length with Base62 encoding (A-Za-z0-9)
- 🚀 **Non-root**: Runs with strict security contexts and minimal privileges
- 📦 **Production-ready**: Multi-arch support (amd64/arm64), signed images, SBOM included

## How It Works

The operator uses Argon2id KDF to derive secrets from a master password with context-based salting:

```
MasterPassword + Context
        ↓
Argon2id(password, salt=context, t=4, m=64MB)
        ↓
Base62 Encoding
        ↓
DerivedSecret
```

This ensures deterministic, secure, and unique secrets for each key.

## Quick Start

### Installation

**Using Helm (Recommended):**

```bash
helm install derived-secret-operator \
  oci://ghcr.io/oleksiyp/charts/derived-secret-operator \
  --version 0.1.6
```

**Using kubectl:**

```bash
kubectl apply -f https://github.com/oleksiyp/derived-secret-operator/releases/latest/download/install.yaml
```

**Using Helmfile:**

```yaml
# helmfile.yaml
repositories: []

releases:
  - name: derived-secret-operator
    chart: oci://ghcr.io/oleksiyp/charts/derived-secret-operator
    version: 0.1.0
    namespace: derived-secret-operator-system
    createNamespace: true
```

```bash
helmfile sync
```

### Create a Master Password

**Note:** If you installed via Helm, a `default` MasterPassword is already created. You can use it directly or create your own with a different name.

```yaml
apiVersion: secrets.oleksiyp.dev/v1alpha1
kind: MasterPassword
metadata:
  name: my-app
spec:
  length: 86
```

### Create a Derived Secret

```yaml
apiVersion: secrets.oleksiyp.dev/v1alpha1
kind: DerivedSecret
metadata:
  name: my-app-secrets
  namespace: default
spec:
  keys:
    DATABASE_PASSWORD:
      type: password
      masterPassword: my-app
    ENCRYPTION_KEY:
      type: encryption-key
      masterPassword: my-app
```

## Getting Started

### Prerequisites
- go version v1.24.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/derived-secret-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/derived-secret-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/derived-secret-operator:tag
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/derived-secret-operator/<tag or branch>/dist/install.yaml
```

### By providing a Helm Chart

1. Build the chart using the optional helm plugin

```sh
kubebuilder edit --plugins=helm/v1-alpha
```

2. See that a chart was generated under 'dist/chart', and users
can obtain this solution from there.

**NOTE:** If you change the project, you need to update the Helm Chart
using the same command above to sync the latest changes. Furthermore,
if you create webhooks, you need to use the above command with
the '--force' flag and manually ensure that any custom configuration
previously added to 'dist/chart/values.yaml' or 'dist/chart/manager/manager.yaml'
is manually re-applied afterwards.

## Releases

### Snapshots (Testing)

Snapshots are built automatically on every push to main:

**Docker Images:**
```bash
# Always latest main branch
docker pull ghcr.io/oleksiyp/derived-secret-operator:edge

# Specific commit
docker pull ghcr.io/oleksiyp/derived-secret-operator:sha-abc123
```

**Helm Charts:**
```bash
# Get version from CI logs (e.g., 0.1.0-dev.abc123)
helm install my-app \
  oci://ghcr.io/oleksiyp/charts/derived-secret-operator \
  --version 0.1.0-dev.abc123
```

### Official Releases

Releases are created manually via GitHub Actions:

1. Go to [GitHub Actions → Release workflow](https://github.com/oleksiyp/derived-secret-operator/actions/workflows/release.yml)
2. Click "Run workflow"
3. Enter version number (e.g., `1.0.0`)
4. Release publishes automatically:
   - Multi-arch Docker images (amd64 + arm64)
   - Helm chart
   - Signed with cosign + SBOM
   - GitHub Release with:
     - Automatic summary of merged PRs
     - Installation instructions
     - install.yaml attachment

**Version Format:** Use [Semantic Versioning](https://semver.org/)
- Major (1.0.0 → 2.0.0): Breaking changes
- Minor (1.0.0 → 1.1.0): New features, backwards compatible
- Patch (1.0.0 → 1.0.1): Bug fixes

**Installation from release:**
```bash
# kubectl
kubectl apply -f https://github.com/oleksiyp/derived-secret-operator/releases/download/v1.0.0/install.yaml

# Helm
helm install derived-secret-operator \
  oci://ghcr.io/oleksiyp/charts/derived-secret-operator \
  --version 1.0.0

# Docker
docker pull ghcr.io/oleksiyp/derived-secret-operator:v1.0.0
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

