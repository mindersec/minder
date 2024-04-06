---
title: Automated Remediations
sidebar_position: 40
---

# Alerts and Automated Remediation in Minder

A profile in Minder offers a comprehensive view of your security posture, encompassing more than just the status report. 
It actively responds to any rules that are not in compliance, taking specific actions. These actions can include the 
creation of alerts for rules that have failed, as well as the execution of remediations to fix the non-compliant 
aspects.

When alerting is turned on in a profile, Minder will open an alert to bring your attention to the non-compliance issue. 
Conversely, when the rule evaluation passes, Minder will automatically close any previously opened alerts related to 
that rule.

When remediation is turned on, Minder also supports the ability to automatically remediate failed rules based on their 
type, i.e., by processing a REST call to enable/disable a non-compliant repository setting or creating a pull request 
with a proposed fix. Note that not all rule types support automatic remediation yet.

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

### Enabling automated remediation in a profile
To activate the remediation feature within a profile, you need to adjust the YAML definition.
Specifically, you should set the remediate parameter to "on":
```yaml
remediate: "on"
```

Enabling remediation at the profile level means that for any rules included in the profile, a remediation action will be
taken for any rule failures.
```yaml
---
version: v1
type: rule-type
name: sample_rule
def:
  remediate:
    type: rest
    rest:
      method: PATCH
      endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}"
      body: |
        { "security_and_analysis": {"secret_scanning": { "status": "enabled" } } }
```
In this example, the `sample_rule` defines a remediation action that performs a PATCH request to an endpoint. This
request will modify the state of the repository ensuring it complies with the rule.

Now, let's see how this works in practice within a profile. Consider the following profile configuration with 
remediation turned on:
```yaml
version: v1
type: profile
name: sample-profile
context:
  provider: github
remediate: "on"
repository:
  - type: sample_rule
    def:
      enabled: true
```
In this profile, all repositories that do not meet the conditions specified in the `sample_rule` will automatically
receive a PATCH request to the specified endpoint. This action will make the repository compliant.
