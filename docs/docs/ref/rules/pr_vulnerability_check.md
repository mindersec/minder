---
title: Known Vulnerabilities
sidebar_position: 60
---

# Known Vulnerabilities Rule

The following rule type is available for known vulnerabilities.

## `pr_vulnerability_check` - Verifies that pull requests do not add dependencies with known vulnerabilities

For every pull request submitted to a repository, this rule will check if the pull request
adds a new dependency with known vulnerabilities based on the [OSV database](https://osv.dev/). If it does, the rule will fail and the
pull request will be rejected or commented on.

### Entity
 - `pull_request`

### Type
 - `pr_vulnerability_check`

### Rule Parameters
 - None

### Rule Definition Options

The `pr_vulnerability_check` rule has the following options:

- `action` (string): The action to take if a vulnerability is found. Valid values are:
    - `review`: Minder will review the PR, suggest changes and mark the PR as changes requested if a vulnerability is found
    - `commit_status`: Minder will comment and suggest changes on the PR if a vulnerability is found. Additionally, Minder
      will set the commit_status of the PR `HEAD` to `failed` to prevent the commit from being merged
    - `comment`: Minder will comment and suggest changes on the PR if a vulnerability is found, but not request changes
    - `summary`: The evaluator engine will add a single summary comment with a table listing the vulnerabilities found
    - `profile_only`: The evaluator engine will merely pass on an error, marking the profile as failed if a vulnerability is found
- `ecosystem_config`: An array of ecosystem configurations to check. Each ecosystem configuration has the following options:
    - `name` (string): The name of the ecosystem to check. Currently `npm`, `go` and `pypi` are supported.
    - `vulnerability_database_type` (string): The kind of vulnerability database to use. Currently only `osv` is supported.
    - `vulnerability_database_endpoint` (string): The endpoint of the vulnerability database to use.
    - `package_repository`: The package repository to use. This is an object with the following options:
        - `url` (string): The URL of the package repository to use. Only the `go` ecosystem uses this option.
    - `sum_repository`: The Go sum repository to use. This is an object with the following options:
        - `url` (string): The URL of the Go sum repository to use.
 
Note that if the `review` action is selected, `minder` will only be able to mark the PR as changes requested if the submitter
is not the same as the Minder identity. If the submitter is the same as the
Minder identity, the PR will only be commented on.

Also note that if `commit_status` action is selected, the PR can only be prevented from merging if the branch protection rules
are set to require a passing commit status.

### Examples

```yaml
- type: pr_vulnerability_check
  def:
  action: review
  ecosystem_config:
  - name: npm
    vulnerability_database_type: osv
    vulnerability_database_endpoint: https://api.osv.dev/v1/query
    package_repository:
      url: https://registry.npmjs.org
  - name: go
    vulnerability_database_type: osv
    vulnerability_database_endpoint: https://api.osv.dev/v1/query
    package_repository:
      url: https://proxy.golang.org
    sum_repository:
      url: https://sum.golang.org
  - name: pypi
    vulnerability_database_type: osv
    vulnerability_database_endpoint: https://api.osv.dev/v1/query
    package_repository:
      url: https://pypi.org/pypi
```
