---
title: Branch protections
sidebar_position: 10
---

# Branch Protection Rules
The following rule type is available for branch protection.

## `branch_protection_allow_deletions` - Whether the branch can be deleted

This rule allows you to allow users with push access to delete matching branches.

### Entity
- `repository`

### Type
- `branch_protection_allow_deletions`

### Rule parameters
The `branch_protection_allow_deletions` rule supports the following parameters:
- `branch (string)` - The name of the branch to check.

### Rule definition options

The `branch_protection_allow_deletions` rule supports the following options:
- `allow_deletions (boolean)` - Allows deletion of the protected branch by anyone with write access to the repository.

## `branch_protection_allow_force_pushes` - Whether force pushes are allowed to the branch

This rule allows you to permit force pushes for all users with push access.

### Entity
- `repository`

### Type
- `branch_protection_allow_force_pushes`

### Rule parameters
The `branch_protection_allow_force_pushes` rule supports the following parameters:
- `branch (string)` - The name of the branch to check.

### Rule definition options

The `branch_protection_allow_force_pushes` rule supports the following options:
- `allow_force_pushes (boolean)` - Permits force pushes to the protected branch by anyone with write access to the repository.

## `branch_protection_allow_fork_syncing` - Whether users can pull changes from upstream when the branch is locked

A locked branch cannot be pulled from.

### Entity
- `repository`

### Type
- `branch_protection_allow_fork_syncing`

### Rule parameters
The `branch_protection_allow_fork_syncing` rule supports the following parameters:
- `branch (string)` - The name of the branch to check.

### Rule definition options

The `branch_protection_allow_fork_syncing` rule supports the following options:
- `allow_fork_syncing (boolean)` - Whether users can pull changes from upstream when the branch is locked. Set to `true` to allow fork syncing. Set to `false` to prevent fork syncing.

## `branch_protection_enabled` - Verifies that a branch has a branch protection rule

You can protect important branches by setting branch protection rules, which define whether
collaborators can delete or force push to the branch and set requirements for any pushes to the branch,
such as passing status checks or a linear commit history.

### Entity
- `repository`

### Type
- `branch_protection_enabled`

### Rule parameters
The `branch_protection_enabled` rule supports the following parameters:
- `branch (string)` - The name of the branch to check.

### Rule definition options

- None

## `branch_protection_enforce_admins` - Whether the protection rules apply to repository administrators

Enforce required status checks for repository administrators.

### Entity
- `repository`

### Type
- `branch_protection_enforce_admins`

### Rule parameters
The `branch_protection_enforce_admins` rule supports the following parameters:
- `branch (string)` - The name of the branch to check.

### Rule definition options
The `branch_protection_enforce_admins` rule supports the following options:
- `enforce_admins (boolean)` - Specifies whether the protection rule applies to repository administrators.

## `branch_protection_lock_branch` - Whether the branch is locked

This rule allows you to set the branch as read-only. Users cannot push to the branch.

### Entity
- `repository`

### Type
- `branch_protection_lock_branch`

### Rule parameters
The `branch_protection_lock_branch` rule supports the following parameters:
- `branch (string)` - The name of the branch to check.

### Rule definition options
The `branch_protection_lock_branch` rule supports the following options:
- `lock_branch (boolean)` - Whether to set the branch as read-only. If this is true, users will not be able to push to the branch.

## `branch_protection_require_conversation_resolution` - Whether PR reviews must be resolved before merging

When enabled, all conversations on code must be resolved before a pull request can be merged into a branch that matches this rule.

### Entity
- `repository`

### Type
- `branch_protection_require_conversation_resolution`

### Rule parameters
The `branch_protection_require_conversation_resolution` rule supports the following parameters:
- `branch (string)` - The name of the branch to check.

### Rule definition options
The `branch_protection_require_conversation_resolution` rule supports the following options:
- `required_conversation_resolution (boolean)` - Requires all conversations on code to be resolved before a pull request can be merged into a branch that matches this rule.

## `branch_protection_require_linear_history` - Whether the branch requires a linear history with no merge commits

This rule allows you to prevent merge commits from being pushed to matching branches.

### Entity
- `repository`

### Type
- `branch_protection_require_linear_history`

### Rule parameters
The `branch_protection_require_linear_history` rule supports the following parameters:
- `branch (string)` - The name of the branch to check.

### Rule definition options
The `branch_protection_require_linear_history` rule supports the following options:
- `required_linear_history (boolean)` - Enforces a linear commit Git history, which prevents anyone from pushing merge commits to a branch.

## `branch_protection_require_pull_request_approving_review_count` - Require a certain number of approving reviews before merging

Each pull request must have a certain number of approving reviews before it can be merged into a matching branch.

### Entity
- `repository`

### Type
- `branch_protection_require_pull_request_approving_review_count`

### Rule parameters
The `branch_protection_require_pull_request_approving_review_count` rule supports the following parameters:
- `branch (string)` - The name of the branch to check.

### Rule definition options
The `branch_protection_require_pull_request_approving_review_count` rule supports the following options:
- `required_approving_review_count (integer)` - Specify the number of reviewers required to approve pull requests. Use a number between 1 and 6 or 0 to not require reviewers.

## `branch_protection_require_pull_request_code_owners_review` - Verifies that a branch requires review from code owners

This rule allows you to require an approved review in pull requests including files with a designated code owner.

### Entity
- `repository`

### Type
- `branch_protection_require_pull_request_code_owners_review`

### Rule parameters
The `branch_protection_require_pull_request_code_owners_review` rule supports the following parameters:
- `branch (string)` - The name of the branch to check.

### Rule definition options
The `branch_protection_require_pull_request_code_owners_review` rule supports the following options:
- `require_code_owner_reviews (boolean)` - Set to true to require an approved review in pull requests including files with a designated code owner.

## `branch_protection_require_pull_request_dismiss_stale_reviews` - Require that new pushes to the branch dismiss old reviews

New reviewable commits pushed to a matching branch will dismiss pull request review approvals.

### Entity
- `repository`

### Type
- `branch_protection_require_pull_request_dismiss_stale_reviews`

### Rule parameters
The `branch_protection_require_pull_request_dismiss_stale_reviews` rule supports the following parameters:
- `branch (string)` - The name of the branch to check.

### Rule definition options
The `branch_protection_require_pull_request_dismiss_stale_reviews` rule supports the following options:
- `dismiss_stale_reviews (boolean)` - Set to true to dismiss approving reviews when someone pushes a new commit.

## `branch_protection_require_pull_request_last_push_approval` - Require that the most recent push to a branch be approved by someone other than the person who pushed it

The most recent push to a branch must be approved by someone other than the person who pushed it.

### Entity
- `repository`

### Type
- `branch_protection_require_pull_request_last_push_approval`

### Rule parameters
The `branch_protection_require_pull_request_last_push_approval` rule supports the following parameters:
- `branch (string)` - The name of the branch to check.

### Rule definition options
The `branch_protection_require_pull_request_last_push_approval` rule supports the following options:
- `require_last_push_approval (boolean)` - Whether the most recent push must be approved by someone other than the person who pushed it.

## `branch_protection_require_pull_requests` - Verifies that a branch requires pull requests

This rule allows you to require that a pull request be opened before merging to a branch.

### Entity
- `repository`

### Type
- `branch_protection_require_pull_requests`

### Rule parameters
The `branch_protection_require_pull_requests` rule supports the following parameters:
- `branch (string)` - The name of the branch to check.

### Rule definition options
The `branch_protection_require_pull_requests` rule supports the following options:
- `required_pull_request_reviews (boolean)` - When enabled, all commits must be made to a non-protected branch and submitted via a pull request before they can be merged into a branch that matches this rule.

## `branch_protection_require_signatures` - Whether commits to the branch must be signed

Commits pushed to matching branches must have verified signatures.

### Entity
- `repository`

### Type
- `branch_protection_require_signatures`

### Rule parameters
The `branch_protection_require_signatures` rule supports the following parameters:
- `branch (string)` - The name of the branch to check.

### Rule definition options
The `branch_protection_require_signatures` rule supports the following options:
- `required_signatures (boolean)` - Specifies whether commits to the branch must be signed.
