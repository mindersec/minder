---
title: Alerting
sidebar_position: 40
---

# Alerts from Minder

Minder issues _alerts_ to notify you when the state of your software supply chain does not meet the criteria that you've defined in your [profile](profile).

Alerts are a core feature of Minder providing you with notifications about the status of your registered
repositories. These alerts automatically open and close based on the evaluation of the rules defined in your profiles.

When a rule fails, Minder opens an alert to bring your attention to the non-compliance issue. Conversely, when the
rule evaluation passes, Minder will automatically close any previously opened alerts related to that rule.

In the alert, you'll be able to see details such as:
* The repository that is affected
* The rule type that failed
* The profile that the rule belongs to
* Guidance on how to remediate and also fix the issue
* Severity of the issue. The severity of the alert is based on what is set in the rule type definition.

### Enabling alerts in a profile
To activate the alert feature within a profile, you need to adjust the YAML definition. 
Specifically, you should set the alert parameter to "on":
```yaml
alert: "on"
```

Enabling alerts at the profile level means that for any rules included in the profile, alerts will be generated for 
any rule failures. For better clarity, consider this rule snippet:
```yaml
---
version: v1
type: rule-type
name: sample_rule
def:
  alert:
      type: security_advisory
      security_advisory:
        severity: "medium"
```
In this example, the `sample_rule` defines an alert action that creates a medium severity security advisory in the 
repository for any non-compliant repositories.

Now, let's see how this works in practice within a profile. Consider the following profile configuration with alerts 
turned on:
```yaml
version: v1
type: profile
name: sample-profile
context:
  provider: github
alert: "on"
repository:
  - type: sample_rule
    def:
      enabled: true
```
In this profile, all repositories that do not meet the conditions specified in the `sample_rule` will automatically
generate security advisories.

## Alert types

Minder supports alerts of type GitHub Security Advisory.

The following is an example of how the alert definition looks like for a give rule type:

```yaml
---
version: v1
type: rule-type
name: artifact_signature
...
def:
  # Defines the configuration for alerting on the rule
  alert:
    type: security_advisory
    security_advisory:
      severity: "medium"
```

## Configuring alerts in profiles

Alerts are configured in the `alert` section of the profile yaml file. The following example shows how to configure
alerts for a profile:

```yaml
---
version: v1
type: profile
name: github-profile
context:
  provider: github
alert: "on"
repository:
  - type: secret_scanning
    def:
      enabled: true
```

The `alert` section can be configured with the following values: `on` (default), `off` and `dry_run`. Dry run would be
useful for testing. In `dry_run` Minder will process the alert conditions and output the resulted REST call, but it
won't execute it.
