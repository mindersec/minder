---
title: GitHub Actions
sidebar_position: 70
---

# GitHub Actions Configuration Rules

There are several rule types that can be used to configure GitHub Actions.

## `github_actions_allowed` - Which actions are allowed to be used

This rule allows you to limit the actions that are allowed to run for a repository.
It is recommended to use the `selected` option for allowed actions, and then
select the actions that are allowed to run.

### Entity
- `repository`

### Type
- `github_actions_allowed`

### Rule parameters
- None

### Rule definition options

The `github_actions_allowed` rule supports the following options:
- `allowed_actions (enum)` - Which actions are allowed to be used
  - `all` - Any action or reusable workflow can be used, regardless of who authored it or where it is defined.
  - `local_only` - Only actions and reusable workflows that are defined in the repository or organization can be used.
  - `selected` - Only the actions and reusable workflows that are explicitly listed are allowed. Use the `allowed_selected_actions` `rule_type` to set the list of allowed actions.

## `allowed_selected_actions` - Verifies that only allowed actions are used

To use this rule, the repository profile for `github_actions_allowed` must
be configured to `selected`.

### Entity
- `repository`

### Type
- `allowed_selected_actions`

### Rule parameters
- None

### Rule definition options

The `allowed_selected_actions` rule supports the following options:
- `github_owner_allowed (boolean)` - Whether GitHub-owned actions are allowed. For example, this includes the actions in the `actions` organization.
- `verified_allowed (boolean)` - Whether actions that are verified by GitHub are allowed.
- `patterns_allowed (boolean)` - Specifies a list of string-matching patterns to allow specific action(s) and reusable workflow(s). Wildcards, tags, and SHAs are allowed.

## `default_workflow_permissions` - Sets the default permissions granted to the `GITHUB_TOKEN` when running workflows

Verifies the default workflow permissions granted to the GITHUB_TOKEN
when running workflows in a repository, as well as if GitHub Actions
can submit approving pull request reviews.

### Entity
- `repository`

### Type
- `default_workflow_permissions`

### Rule parameters
- None

### Rule definition options

The `default_workflow_permissions` rule supports the following options:
- `default_workflow_permissions (boolean)` - Whether GitHub-owned actions are allowed. For example, this includes the actions in the `actions` organization.
- `can_approve_pull_request_reviews (boolean)` - Whether the `GITHUB_TOKEN` can approve pull request reviews.

## `actions_check_pinned_tags` - Verifies that any actions use pinned tags

Verifies that actions use pinned tags as opposed to floating tags.

### Entity
- `repository`

### Type
- `actions_check_pinned_tags`

### Rule parameters
- None

### Rule definition options
- None
