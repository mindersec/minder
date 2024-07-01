---
title: Artifact signature
sidebar_position: 90
---

# Artifact signature verification

The following rule type is available for checking that an artifact has a valid signature
and its provenance conforms to a policy.

## `artifact_signature` - Verifies that an artifact has a valid signature

This rule allows you to verify that an artifact was signed and that the signature is valid.

## Entity
- `artifact`

## Type
- `artifact_signature`

## Rule Parameters
- `tags` - the tags that should be checked for signatures. If not specified, all tags will be checked. If specified, the artifact must be tagged with all of the specified tags in order to be checked.
- `tags_regex` - a regular expression specifying the tags that should be checked for signatures. If not specified, all tags will be checked. If specified, the artifact must be tagged with a tag that matches the regular expression in order to be checked.
- `name` - the name of the artifact that should be checked for signatures. If not specified, all artifacts will be checked.
 
It is an error to specify both `tags` and `tags_regex`.

## Rule Definition Options

The `artifact_signature` rule has the following options:

- `is_signed` (bool): Whether the artifact is signed
- `is_verified` (bool): Whether the artifact's signature could be verified
- `repository` (string): The repository that the artifact was built from
- `branch` (string): The branch that the artifact was built from
- `signer_identity` (string): The identity of the signer of the artifact, e.g. a workflow name like `docker-image-build-push.yml` for GitHub workflow signatures or an email address
- `runner_environment` (string): The environment that the artifact was built in, i.e. hosted-runner or self-hosted-runner. Set to `github-hosted` to check for artifacts built on a GitHub-hosted runner.
- `cert_issuer` (string): The issuer of the certificate used to sign the artifact, i.e. `https://token.actions.githubusercontent.com` for GitHub Actions
