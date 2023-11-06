---
title: Activity Score
sidebar_position: 20
---

# Activity Score Threshold Policy

The following rule type is available for [Trusty](https://www.trustypkg.dev/) activity score threshold.

## `pr_trusty_check` - Verifies that pull requests do not add any dependencies with low Trusty scores

This rule allows you to monitor new Pull requests for newly added dependencies with low
[Trusty](https://www.trustypkg.dev/) scores.
For every pull request submitted to a repository, this rule will check if the pull request adds a new dependency with
a low Trusty score. If a dependency with a low score is added, the PR will be commented on.

## Entity
- `pull_request`

## Type
- `pr_trusty_check`

## Rule Parameters
- None

## Rule Definition Options

The `pr_trusty_check` rule has the following options:

- `action` (string): The action to take if a package with a low score is found. Valid values are:
  - `summary`: The evaluator engine will add a single summary comment with a table listing the packages with low scores found
  - `profile_only`: The evaluator engine will merely pass on an error, marking the profile as failed if a packages with low scores is found
- `ecosystem_config`: An array of ecosystem configurations to check. Each ecosystem configuration has the following options:
  - `name` (string): The name of the ecosystem to check. Currently `npm` and `pypi` are supported.
  - `pi_threshold` (number): The minimum Trusty score for a dependency to be considered safe.
