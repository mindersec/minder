---
title: Trusty Score
sidebar_position: 20
---

# Trusty Score Threshold Rule

The following rule type is available for [Trusty](https://www.trustypkg.dev/) score threshold.

## `pr_trusty_check` - Verifies that pull requests do not add any dependencies with Trusty scores below a certain threshold 

This rule allows you to monitor new pull requests for newly added dependencies with low
[Trusty](https://www.trustypkg.dev/) scores.
For every pull request submitted to a repository, this rule will check if the pull request adds a new dependency with
a Trusty score below a threshold that you define. If a dependency with a low score is added, the PR will be commented on.

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
  - `review`: The trusty evaluator will add a review asking for changes when problematic dependencies are found. Use the review action to block any pull requests introducing dependencies that break the policy established defined by the rule.
- `ecosystem_config`: An array of ecosystem configurations to check. Each ecosystem configuration has the following options:
  - `name` (string): The name of the ecosystem to check. Currently `npm` and `pypi` are supported.
  - `score` (number): The minimum Trusty score for a dependency to be considered safe.
  - `provenance` (number): Minimum provenance score to consider a package's proof of origin satisfactory.
  - `activity` (number): Minimum activity score to consider a package as active.
  - `allow_malicious` (boolean): Don't raise an error when a PR introduces dependencies known to be malicious (not recommended)
  - `allow_deprecated` (boolean): Don't block when a pull request introduces dependencies marked as deprectaed upstream.
