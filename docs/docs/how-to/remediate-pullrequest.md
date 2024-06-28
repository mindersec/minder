---
title: Automatic remediation via Pull Request
sidebar_position: 65
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

# Creating a pull request for automatic remediation

## Prerequisites

* The `minder` CLI application
* A Minder account
* An enrolled Provider (e.g., GitHub) and registered repositories

## Create a rule type that has support for pull request automatic remediation

The pull request automatic remediation feature provides the functionality to fix a failed rule type by creating a pull request. 

This feature is only available for rule types that support it. To find out if a rule type supports it, check the
`remediate` section in their `<alert-type>.yaml` file. It should have the `pull_request` section defined like below:

```yaml
version: v1
type: rule-type
...
  remediate:
    type: pull_request
...
```

In this example, we will use a rule type that checks if a repository has Dependabot enabled. If it's not enabled, Minder
will create a pull request that enables Dependabot. The rule type is called `dependabot_configured.yaml` and is one of 
the reference rule types provided by the Minder team.

Fetch all the reference rules by cloning the [minder-rules-and-profiles repository](https://github.com/stacklok/minder-rules-and-profiles).

```bash
git clone https://github.com/stacklok/minder-rules-and-profiles.git
```

In that directory, you can find all the reference rules and profiles.

```bash
cd minder-rules-and-profiles
```

Create the `dependabot_configured` rule type in Minder:

```bash
minder ruletype create -f rule-types/github/dependabot_configured.yaml
```

## Create a profile
Next, create a profile that applies the rule to all registered repositories.

Create a new file called `profile.yaml`.

Based on your source code language, paste the following profile definition into the newly created file.

<Tabs>
<TabItem value="go" label="Go" default>

```yaml
---
version: v1
type: profile
name: dependabot-profile
context:
  provider: github
alert: "on"
remediate: "on"
repository:
  - type: dependabot_configured
    def:
      package_ecosystem: gomod
      schedule_interval: weekly
      apply_if_file: go.mod
```

</TabItem>
<TabItem value="npm" label="NPM">

```yaml
---
version: v1
type: profile
name: dependabot-profile
context:
  provider: github
alert: "on"
remediate: "on"
repository:
  - type: dependabot_configured
    def:
      package_ecosystem: npm
      schedule_interval: weekly
      apply_if_file: package.json
```
</TabItem>
</Tabs>

Create the profile in Minder:

```bash
minder profile create -f profile.yaml
```

Once the profile is created, Minder will monitor all of your registered repositories matching the expected ecosystem,
i.e., Go, NPM, etc.

If a repository does not have Dependabot enabled, Minder will create a pull request with the necessary configuration
to enable it. Alongside the pull request, Minder will also create a Security Advisory alert that will be present until the issue
is resolved.

Alerts are complementary to the remediation feature. If you have both `alert` and `remediation` enabled for a profile,
Minder will attempt to remediate it first. If the remediation fails, Minder will create an alert. If the remediation
succeeds, Minder will close any previously opened alerts related to that rule.

## Limitations

* The pull request automatic remediation feature is only available for rule types that support it.
* There's no support for creating pull requests that modify the content of existing files yet.
* The created pull request should be closed manually if the issue is resolved through other means. The profile status and any related alerts will be updated/closed automatically.
