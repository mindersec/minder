---
title: Minder profiles
sidebar_position: 20
---

# Profiles in Minder

Profiles in Minder are the foundation of your compliance monitoring strategy, allowing you to group and manage
rules for various entity types, such as repositories, pull requests, and artifacts, across your registered GitHub
repositories.

Profiles are defined in a YAML file and can be easily configured to meet your compliance requirements.

## Rules

Each rule type within a profile is evaluated against your repositories that are registered with Minder.

The available entity rule type groups are `repository`, `pull_request`, and `artifact`.

Each rule type group has a set of rules that can be configured individually.

## Actions

Minder supports the ability to perform actions based on the evaluation of a rule such as creating alerts
and automatically remediating non-compliant rules.

When a rule fails, Minder can open an alert to bring your attention to the non-compliance issue. Conversely, when the
rule evaluation passes, Minder will automatically close any previously opened alerts related to that rule.

Minder also supports the ability to automatically remediate failed rules based on their type, i.e., by processing a
REST call to enable/disable a non-compliant repository setting or creating a pull request with a proposed fix. Note
that not all rule types support automatic remediation yet.

Both alerts and remediations are configured in the profile YAML file under `alerts` (Default: `on`)
and `remediate` (Default: `off`).

## Example profile

Here's a profile which has a `repository` entity group with a rule of type `secret_scanning` set to `enabled: true` and
its `alert` and `remediate` features are both turned `on`:

    ```yaml
    ---
    version: v1
    type: profile
    name: github-profile
    context:
      provider: github
    alert: "on"
    remediate: "on"
    repository:
      - type: secret_scanning
        def:
          enabled: true
    ```
