# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of derived-secret-operator
- Kubernetes operator for deriving secrets from master passwords using Argon2id KDF
- MasterPassword and DerivedSecret custom resources
- Automatic secret generation and rotation
- Helm chart for easy deployment
- Multi-architecture container images (amd64, arm64)
- Comprehensive CI/CD workflows with GitHub Actions
- End-to-end testing suite
