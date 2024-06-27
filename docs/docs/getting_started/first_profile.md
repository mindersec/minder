---
title: Creating your first profile
sidebar_position: 60
---

# Creating your first profile

Minder uses [profiles](../how-to/create_profile.md) to specify common,
consistent configuration which should be enforced on all registered
repositories.  In this tutorial, you will register a GitHub repository and
create a profile that indicates whether secret scanning is enabled on the
registered repositories.

## Prerequisites

Before you can create a profile, you should [register repositories](register_repos).

## Creating and applying profiles

A profile is a set of rules that you apply to your registered repositories.
Before creating a profile, you need to ensure that all desired rule_types have been created in Minder.

Start by creating a rule that checks if secret scanning is enabled and creates
a security advisory alert if secret scanning is not enabled.  
This is a reference rule provided by the Minder team in the [minder-rules-and-profiles repository](https://github.com/stacklok/minder-rules-and-profiles).

For this exercise, we're going to download just the `secret_scanning.yaml`
rule, and then use `minder ruletype create` to define the secret scanning rule.

```bash
curl -LO https://raw.githubusercontent.com/stacklok/minder-rules-and-profiles/main/rule-types/github/secret_scanning.yaml
```

Once you've downloaded the rule definition, you can create it in your Minder account:

```bash
minder ruletype create -f secret_scanning.yaml
```

Next, create a profile that applies the secret scanning rule.

Create a new file called `profile.yaml`.
Paste the following profile definition into the newly created file.

```yaml
---
version: v1
type: profile
name: github-profile
context:
  provider: github
alert: "on"
remediate: "off"
repository:
  - type: secret_scanning
    def:
      enabled: true
```

Create the profile in Minder:
```
minder profile create -f profile.yaml
```

Check the status of the profile:
```
minder profile status list --name github-profile
```
If all registered repositories have secret scanning enabled, you will see the `OVERALL STATUS` is `Success`, otherwise the 
overall status is `Failure`.

```
+--------------------------------------+----------------+----------------+----------------------+
|                  ID                  |      NAME      | OVERALL STATUS |     LAST UPDATED     |
+--------------------------------------+----------------+----------------+----------------------+
| 1abcae55-5eb8-4d9e-847c-18e605fbc1cc | github-profile |    Success     | 2023-11-06T17:42:04Z |
+--------------------------------------+----------------+----------------+----------------------+
```

If secret scanning is not enabled, you will see `Failure` instead of `Success`.


See a detailed view of which repositories satisfy the secret scanning rule:
```
minder profile status list --name github-profile --detailed
```

## Viewing alerts

Disable secret scanning in one of the registered repositories, by following 
[these instructions provided by GitHub](https://docs.github.com/en/code-security/secret-scanning/configuring-secret-scanning-for-your-repositories).

Navigate to the repository on GitHub, click on the Security tab and view the Security Advisories.  
Notice that there is a new advisory titled `minder: profile github-profile failed with rule secret_scanning`.

Enable secret scanning in the same registered repository, by following
[these instructions provided by GitHub](https://docs.github.com/en/code-security/secret-scanning/configuring-secret-scanning-for-your-repositories).

Navigate to the repository on GitHub, click on the Security tab and view the Security Advisories.
Notice that the advisory titled `minder: profile github-profile failed with rule secret_scanning` is now closed.

## Delete registered repositories

If you want to stop monitoring a repository, you can delete it from Minder by using the `repo delete` command:
```bash
minder repo delete --name ${REPO_NAME}
```
where `$REPO_NAME` is the fully-qualified name (`owner/name`) of the repository you wish to delete, for example `testorg/testrepo`.

This will delete the repository from Minder and remove the webhook from the repository. 
