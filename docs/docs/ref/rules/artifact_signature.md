---
title: Artifact Signature
sidebar_position: 90
---

# Artifact signature verification

The following rule type is available for checking that an artifact has a valid signature.

## `artifact_signature` - Verifies that an artifact has a valid signature

This rule allows you to verify that an artifact was signed and that the signature is valid.

## Entity
- `artifact`

## Type
- `artifact_signature`

## Rule Parameters
- `tags` - the tags that should be checked for signatures. If not specified, all tags will be checked. If specified, the artifact must be tagged with all of the specified tags in order to be checked.
- `name` - the name of the artifact that should be checked for signatures. If not specified, all artifacts will be checked.

## Rule Definition Options

The `artifact_signature` rule has the following options:

- `is_signed` (bool): Whether the artifact is signed
- `is_verified` (bool): Whether the artifact's signature could be verified
- `is_bundle_verified` (bool): Whether the artifact's bundle signature could be verified
