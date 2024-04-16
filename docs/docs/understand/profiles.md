---
title: Profiles and Rules
sidebar_position: 10
---

# Profiles in Minder

A _profile_ defines your security policies that you want to apply to your software supply chain. Profiles contain rules that query data in a [provider](providers), and specifies whether Minder will issue [alerts](alerts) or perform automatic [remediations](remediations) when an entity is not in compliance with the policy.

Profiles in Minder allow you to group and manage
rules for various entity types, such as repositories, pull requests, and artifacts, across your registered GitHub
repositories.

The anatomy of a profile is the profile itself, which outlines the rules to be
checked, the rule types, and the evaluation engine. Profiles are defined in a YAML file and can be configured to meet your compliance requirements.
As of time of writing, Minder supports the following evaluation engines:

* **[Rego](https://www.openpolicyagent.org/docs/latest/policy-language/)** Open Policy Agents's native query language, Rego.
* **[JQ](https://jqlang.github.io/jq/)** - a lightweight and flexible command-line JSON processor.

Each engine is designed to be extensible, allowing you to integrate your own
logic and processes.

Stacklok has published [a set of example profiles on GitHub](https://github.com/stacklok/minder-rules-and-profiles/tree/main/profiles/github); see [Managing Profiles](../how-to/manage_profiles.md) for more details on how to set up profiles and rule types.

## Rules

Each rule type within a profile is evaluated against your repositories that are registered with Minder.

The available entity rule type groups are `repository`, `pull_request`, and `artifact`.

Each rule type group has a set of rules that can be configured individually.

The available properties for each rule type can be found in their YAML file under the `def.rule_schema.properties` section.

For example, the `secret_scanning` rule type has the following `enabled` property:

```yaml
---
version: v1
type: rule-type
name: secret_scanning
---
def:
  # Defines the schema for writing a rule with this rule being checked
  rule_schema:
    properties:
      enabled:
        type: boolean
        default: true
```

You can find a list of available rules in the [https://github.com/stacklok/minder-rules-and-profiles](https://github.com/stacklok/minder-rules-and-profiles/tree/main/rule-types/github) repository.

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

Here's a profile which has a single rule for each entity group and its `alert` and `remediate` features are both 
turned `on`:

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
  - type: dependabot_configured
    def:
      package_ecosystem: gomod
      schedule_interval: daily
      apply_if_file: go.mod
artifact:
  - type: artifact_signature
    params:
      tags: [latest]
      name: your-artifact-name
    def:
      is_signed: true
      is_verified: true
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
