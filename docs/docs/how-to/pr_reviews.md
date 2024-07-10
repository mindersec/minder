---
title: Enabling pull request reviews
sidebar_position: 50
---

import Tabs from '@theme/Tabs'; import TabItem from '@theme/TabItem';

## Prerequisites

- The `minder` CLI application
- A Minder account with
  [at least `editor` permission](../user_management/user_roles.md)
- An enrolled Provider (e.g., GitHub) and registered repositories

## Create the PR vulnerability check rule

Start by creating a rule that checks if a pull request adds a new dependency
with known vulnerabilities. If it does, Minder will review the PR and suggest
changes.

Note that Minder is only able to review a PR if it's running under a different
user than the one that created the PR. If the PR is created by the same user,
Minder only provides a comment with the vulnerability information. An
alternative is to use the `commit-status` action instead of `review` where
Minder will set the commit status to `failure` if the PR introduces a new
vulnerability which can then be used to block the PR. This requires an
additional step though, where the repo needs to require the
`minder.stacklok.dev/pr-vulncheck` status to be passing.

This is a reference rule provider by the Minder team.

Fetch all the reference rules by cloning the
[minder-rules-and-profiles repository](https://github.com/stacklok/minder-rules-and-profiles).

```
git clone https://github.com/stacklok/minder-rules-and-profiles.git
```

In that directory you can find all the reference rules and profiles.

```
cd minder-rules-and-profiles
```

Create the `pr_vulnerability_check` rule type in Minder:

```
minder ruletype create -f rule-types/github/pr_vulnerability_check.yaml
```

## Create a profile

Next, create a profile that applies the rule to all registered repositories.

Create a new file called `profile.yaml`. Based on your source code language,
paste the following profile definition into the newly created file.

<Tabs>
<TabItem value="go" label="Go" default>

```yaml
---
version: v1
type: profile
name: pr-review-profile
context:
  provider: github
alert: "on"
remediate: "off"
pull_request:
  - type: pr_vulnerability_check
    def:
      action: review
      ecosystem_config:
        - name: go
          vulnerability_database_type: osv
          vulnerability_database_endpoint: https://api.osv.dev/v1/query
          package_repository:
            url: https://proxy.golang.org
          sum_repository:
            url: https://sum.golang.org
```

</TabItem>
<TabItem value="npm" label="NPM">

```yaml
---
version: v1
type: profile
name: pr-review-profile
context:
  provider: github
alert: "on"
remediate: "off"
pull_request:
  - type: pr_vulnerability_check
    def:
      action: review
      ecosystem_config:
        - name: npm
          vulnerability_database_type: osv
          vulnerability_database_endpoint: https://api.osv.dev/v1/query
          package_repository:
            url: https://registry.npmjs.org
```

</TabItem>
<TabItem value="pypi" label="PyPI">

```yaml
---
version: v1
type: profile
name: pr-review-profile
context:
  provider: github
alert: "on"
remediate: "off"
pull_request:
  - type: pr_vulnerability_check
    def:
      action: review
      ecosystem_config:
        - name: pypi
          vulnerability_database_type: osv
          vulnerability_database_endpoint: https://api.osv.dev/v1/query
          package_repository:
            url: https://pypi.org/pypi
```

</TabItem>
</Tabs>

Create the profile in Minder:

```
minder profile create -f profile.yaml
```

Once the profile is created, Minder will monitor any pull requests to the
registered repositories. If a pull request brings in a dependency with a known
vulnerability, then Minder will add a review to the pull request and suggest
changes.

Alerts are complementary to the remediation feature. If you have both `alert`
and `remediation` enabled for a profile, Minder will attempt to remediate it
first. If the remediation fails, Minder will create an alert. If the remediation
succeeds, Minder will close any previously opened alerts related to that rule.
