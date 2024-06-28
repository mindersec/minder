---
title: Presence of a license file
sidebar_position: 80
---

# Presence of a License File Rule

The following rule type is available for verifying if a license file is present and it is of a certain type.

## `license` - Verifies if there is a license file of a given type present in the repository

This rule allows you to monitor if a license file is present in the repository and if its license type complies with
the configured license type in your profile.

### Entity
- `repository`

### Type
- `license`

### Rule parameters
- None

### Rule definition options

The `license` rule supports the following options:
- `license_filename (string)` - The license filename to look for.
    - Example: `LICENSE`, `LICENSE.txt`, `LICENSE.md`, etc.
- `license_type (string)` - The license type to look for in `license_filename`.
    - Example: `MIT`, `Apache`, etc. See [SPDX License List](https://spdx.org/licenses/) for a list of license types. Leave `""` to only check for the presence of the file.
