---
title: Automatic remediations
sidebar_position: 70
---

# Automatic Remediations in Minder

Minder can perform _automatic remediation_ for many rules in an attempt to resolve problems in your software supply chain, and bring your resources into compliance with your [profile](profiles.md).

The steps to take during automatic remediation are defined within the rule itself and can perform actions like sending a REST call to an endpoint to change configuration, or creating a pull request with a proposed fix.

For example, if you have a rule in your profile that specifies that Secret Scanning should be enabled, and you have enabled automatic remediation in your profile, then Minder will attempt to turn Secret Scanning on in any repositories where it is not enabled.

### Enabling remediations in a profile
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

## Limitations

Some rule types do not support automatic remediations, due to platform limitations. For example, it may be possible to query the status of a repository configuration, but there may not be an API to _change_ the configuration. In such case, a rule type could detect problems but would not be able to remediate.

To identify which rule types support remediation, you can run:

```bash
minder ruletype list -oyaml
```

This will show all the rule types; a rule type with a `remediate` attribute supports automatic remediation.

Furthermore, remediations that open a pull request such as the `dependabot` rule type only attempt to replace the target file, overwriting its contents. This means that if you want to keep the current changes, you need to merge the contents manually.
