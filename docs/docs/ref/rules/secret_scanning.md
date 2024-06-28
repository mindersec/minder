---
title: Secret scanning
sidebar_position: 50
---

# Secret Scanning Rule

The following rule type is available for secret scanning.

## `secret_scanning` - Verifies that secret scanning is enabled for a given repository

Secret scanning is a feature that scans repositories for secrets and alerts
the repository owner when a secret is found. To enable this feature in GitHub,
you must enable it in the repository settings.

### Entity
- `repository`

### Type
- `secret_scanning`

### Rule parameters
- None

### Rule definition options

The `secret_scanning` rule supports the following options:
- `enabled (boolean)` - Whether secret scanning should be enabled for a given repository.
