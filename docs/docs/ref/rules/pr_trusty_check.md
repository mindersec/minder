---
title: Stacklok Insight check
sidebar_position: 20
---

# Stacklok Insight Rule

The following rule type is available to check dependency risk with
[Stacklok Insight](https://insight.stacklok.com/).

## `pr_trusty_check` - Verifies that pull requests do not add any dependencies with risk indicators from Stacklok Insight

This rule allows you to monitor new pull requests for newly added dependencies
with risk indicators from [Stacklok Insight](https://insight.stacklok.com/). For
every pull request submitted to a repository, this rule will check any software
dependencies for the supported ecosystems and flag any problems found with them.
Based on the Stacklok Insight data, Minder can block the PR or mark the policy
as failed.

## Entity

- `pull_request`

## Type

- `pr_trusty_check`

## Rule Parameters

- None

## Rule Definition Options

The `pr_trusty_check` rule supports the following options:

- `action` (string): The action to take if a risky package is found. Valid
  values are:
  - `summary`: The evaluator engine will add a single summary comment with a
    table listing risky packages found
  - `profile_only`: The evaluator engine will merely pass on an error, marking
    the profile as failed if a risky package is found
  - `review`: The trusty evaluator will add a review asking for changes when
    problematic dependencies are found. Use the review action to block any pull
    requests introducing dependencies that break the policy established defined
    by the rule.
- `ecosystem_config`: An array of ecosystem configurations to check. Each
  ecosystem configuration has the following options:
  - `name` (string): The name of the ecosystem to check. Currently `npm` and
    `pypi` are supported.
  - `score (integer)`: DEPRECATED - this score is deprecated and only remains
    for backward compatibility. It always returns a value of `0`. We recommend
    setting this option to `0` and using the other options to control this
    rule's behavior.
  - `provenance` (number): Minimum provenance score to consider a package's
    proof of origin satisfactory.
  - `activity` (number): Minimum activity score to consider a package as active.
  - `allow_malicious` (boolean): Don't raise an error when a PR introduces
    dependencies known to be malicious (not recommended)
  - `allow_deprecated` (boolean): Don't block when a pull request introduces
    dependencies marked as deprecated upstream.
