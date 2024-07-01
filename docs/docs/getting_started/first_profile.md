---
title: Creating your first profile
sidebar_position: 60
---

# Creating your first profile

Once you have registered repositories, you can create a profile that specifies the common, consistent configuration that you expect your your repositories to comply with. 

## Prerequisites

Before you can create a profile, you should [register repositories](register_repos).

## Creating a profile

A profile is composed of a set of one or more rule types, each of which scan for an individual piece of configuration within a repository.

For example, you may have a profile that describes your organization's security best practices, and this profile may have two rule types: GitHub secret scanning should be enabled, and secret push protection should be enabled.

In this example, Minder will scan the repositories that you've registered and identify any repositories that do not have secret scanning enabled, and those that do not have secret push protection enabled.

### Adding rule types

In Minder, the rule type configuration is completely flexible, and you can author them yourself. Because of this, your Minder organization does not have any rule types configured when you create it. You need to configure some, and you can upload some of Minder's already created rules from the [minder-rules-and-profiles repository](https://github.com/stacklok/minder-rules-and-profiles).

For example, to add a rule type that ensures that secret scanning is enabled, and one that ensures that secret push protection is enabled, you can download the [secret_scanning.yaml](https://github.com/stacklok/minder-rules-and-profiles/blob/main/rule-types/github/secret_scanning.yaml) and [secret_push_protection.yaml](https://github.com/stacklok/minder-rules-and-profiles/blob/main/rule-types/github/secret_push_protection.yaml) rule types.

```bash
curl -LO https://raw.githubusercontent.com/stacklok/minder-rules-and-profiles/main/rule-types/github/secret_scanning.yaml
curl -LO https://raw.githubusercontent.com/stacklok/minder-rules-and-profiles/main/rule-types/github/secret_push_protection.yaml
```

Once you've downloaded the rule type configuration from GitHub, you can upload them to your Minder organization.

```bash
minder ruletype create -f secret_scanning.yaml
minder ruletype create -f secret_push_protection.yaml
```

### Creating a profile

Once you have added the rule type definitions to Minder, you can create a profile that uses them to scan your repositories.

Like rules, profiles are defined in YAML. They specify the rules that apply to your organization, whether you want to be alerted with GitHub Security Advisories, and whether you want Minder to try to automatically remediate problems when they're found.

To create a profile that checks for secret scanning and secret push protection in your repositories, create a file called `my_profile.yaml`:

```yaml
---
version: v1
type: profile
name: my_profile
context:
  provider: github
alert: "on"
remediate: "off"
repository:
  - type: secret_scanning
    def:
      enabled: true
  - type: secret_push_protection
    def:
      enabled: true
```

Then upload the profile configuration to Minder:

```
minder profile create -f my_profile.yaml
```



Check the status of the profile:
```
minder profile status list --name my_profile
```
If all registered repositories have secret scanning enabled, you will see the `OVERALL STATUS` is `Success`, otherwise the 
overall status is `Failure`.

```
+--------------------------------------+------------+----------------+----------------------+
|                  ID                  |    NAME    | OVERALL STATUS |     LAST UPDATED     |
+--------------------------------------+------------+----------------+----------------------+
| 1abcae55-5eb8-4d9e-847c-18e605fbc1cc | my_profile |    Success     | 2023-11-06T17:42:04Z |
+--------------------------------------+------------+----------------+----------------------+
```

If secret scanning is not enabled, you will see `Failure` instead of `Success`.


See a detailed view of which repositories satisfy the secret scanning rule:
```
minder profile status list --name my_profile --detailed
```
