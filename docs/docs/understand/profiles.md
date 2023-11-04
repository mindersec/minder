---
title: Profiles
sidebar_position: 20
---

# Profiles in Minder

Profiles in Minder are the foundation of your compliance monitoring strategy, allowing you to group and manage
rules for various entity types, such as repositories, pull requests, and artifacts, across your registered GitHub
repositories.

Profiles are defined in a YAML file and can be easily configured to meet your compliance requirements.

You can find a list of available profiles in the [https://github.com/stacklok/minder-rules-and-profiles](https://github.com/stacklok/minder-rules-and-profiles/tree/main/profiles/github) repository.

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
  - type: secret_scanning
    def:
      enabled: true
artifact:
  - type: artifact_signature
    params:
      tags: [main]
      name: my-artifact
    def:
      is_signed: true
      is_verified: true
      is_bundle_verified: true
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
```
