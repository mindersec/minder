---
title: Setting up a profile for GitHub Security Advisories
sidebar_position: 70
---

# Setting up a profile for GitHub Security Advisories

## Prerequisites

* The `minder` CLI application
* A Minder account
* An enrolled Provider (e.g., GitHub) and registered repositories

## Create a rule type that you want to be alerted on

The `alert` feature is available for all rule types that have the `alert` section defined in their `<alert-type>.yaml`
file. Alerts are a core feature of Minder providing you with notifications about the status of your registered
repositories using GitHub Security Advisories. These Security Advisories automatically open and close based on the evaluation of the rules defined in your profiles.

When a rule fails, Minder opens an alert to bring your attention to the non-compliance issue. Conversely, when the
rule evaluation passes, Minder will automatically close any previously opened alerts related to that rule.

In this example, we will use a rule type that checks if a repository has a LICENSE file present. If there's no file
present, Minder will create an alert notifying the owner of the repository. The rule type is called `license.yaml` and
is one of the reference rule types provided by the Minder team. Details, such as the severity of the alert are defined
in the `alert` section of the rule type definition.

Fetch all the reference rules by cloning the [minder-rules-and-profiles repository](https://github.com/stacklok/minder-rules-and-profiles).

```bash
git clone https://github.com/stacklok/minder-rules-and-profiles.git
```

In that directory, you can find all the reference rules and profiles.

```bash
cd minder-rules-and-profiles
```

Create the `license` rule type in Minder:

```bash
minder ruletype create -f rule-types/github/license.yaml
```

## Create a profile
Next, create a profile that applies the rule to all registered repositories.

Create a new file called `profile.yaml` using the following profile definition and enable alerting by setting `alert`
to `on` (default). The other available values are `off` and `dry_run`.

```yaml
---
version: v1
type: profile
name: license-profile
context:
  provider: github
alert: "on"
repository:
  - type: license
    def:
      license_filename: LICENSE
      license_type: ""
```

Create the profile in Minder:

```bash
minder profile create -f profile.yaml
```

Once the profile is created, Minder will monitor all of your registered repositories for the presence of the `LICENSE`
file.

If a repository does not have a `LICENSE` file available, Minder will create an alert of type Security Advisory providing
additional details such as the profile and rule that triggered the alert and guidelines on how to resolve the issue.

Once a `LICENSE` file is added to the repository, Minder will automatically close the alert.

Alerts are complementary to the remediation feature. If you have both `alert` and `remediation` enabled for a profile,
Minder will attempt to remediate it first. If the remediation fails, Minder will create an alert. If the remediation
succeeds, Minder will close any previously opened alerts related to that rule.

## Limitations

* Currently, the only supported alert type is GitHub Security Advisory. More alert types will be added in the future.
* Alerts are only available for rules that have the `alert` section defined in their `<alert-type>.yaml` file.
