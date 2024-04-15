---
title: Setting up a Profile for Auto-remediation
sidebar_position: 60
---

# Setting up a Profile for Autoremediation

## Prerequisites

* The `minder` CLI application
* A Minder account
* An enrolled Provider (e.g., GitHub) and registered repositories

## Create a rule type that you want to use auto-remediation on

The `remediate` feature is available for all rule types that have the `remediate` section defined in their
`<alert-type>.yaml` file. When the `remediate` feature is turned `on`, Minder will try to automatically remediate failed
rules based on their type, i.e., by processing a REST call to enable/disable a non-compliant repository setting or by
creating a pull request with a proposed fix.

In this example, we will use a rule type that checks if a repository allows having force pushes on their main branch,
which is considered a security risk. If their setting allows for force pushes, Minder will automatically remediate it
and disable it. 

The rule type is called `branch_protection_allow_force_pushes.yaml` and is one of the reference rule types provided by
the Minder team.

Fetch all the reference rules by cloning the [minder-rules-and-profiles repository](https://github.com/stacklok/minder-rules-and-profiles).

```bash
git clone https://github.com/stacklok/minder-rules-and-profiles.git
```

In that directory, you can find all the reference rules and profiles.

```bash
cd minder-rules-and-profiles
```

Create the `branch_protection_allow_force_pushes` rule type in Minder:

```bash
minder ruletype create -f rule-types/github/branch_protection_allow_force_pushes.yaml
```

## Create a profile
Next, create a profile that applies the rule to all registered repositories.

Create a new file called `profile.yaml` using the following profile definition and enable auto remediation by setting
`remediate` to `on`. The other available values are `off`(default) and `dry_run`.

```yaml
---
version: v1
type: profile
name: disable-force-push-profile
context:
  provider: github
remediate: "on"
repository:
  - type: branch_protection_allow_force_pushes
    params:
      branch: main
    def:
      allow_force_pushes: false
```

Create the profile in Minder:

```bash
minder profile create -f profile.yaml
```

Once the profile is created, Minder will monitor if the `allow_force_pushes` setting on all of your registered
repositories is set to `false`. If the setting is set to `true`, Minder will automatically remediate it by disabling it
and will make sure to keep it that way until the profile is deleted.

Alerts are complementary to the remediation feature. If you have both `alert` and `remediation` enabled for a profile,
Minder will attempt to remediate it first. If the remediation fails, Minder will create an alert. If the remediation
succeeds, Minder will close any previously opened alerts related to that rule.

## Limitations

* The auto remediation feature is only available for rule types that support it, i.e., have the `remediate` section defined in their `<alert-type>.yaml` file.
