---
title: Enabling Secret Scanning
sidebar_position: 40
---

# Enabling Secret Scanning using Minder Profiles

Minder uses [profiles](../how-to/create_profile.md) to specify common,
consistent configuration which should be enforced on all registered
repositories.  In this tutorial, you will register a GitHub repository and
create a profile that indicates whether secret scanning is enabled on the
registered repositories.

## Prerequisites

* [The `minder` CLI application](./install_cli.md)
* [A Minder account](./login.md)
* [An enrolled GitHub token](./login.md#enrolling-the-github-provider) that is either an Owner in the organization or an Admin on the repositories

## Register repositories

Once you have enrolled a provider, you can register repositories from that provider.

```bash
minder repo register --provider github
```

This command will show a list of the public repositories available for
registration.  Navigate through the repositories using the arrow keys and
select one or more repositories for registration using the space key.  Press
enter once you have selected all the desired repositories.

You can also register a repository (or set of repositories) by name:

```bash
minder repo register --provider github --repo "owner:repo1,owner:repo2"
```

You can see the list of repositories registered in Minder with the `repo list` command:
```bash
minder repo list --provider github
```

## Creating and applying profiles

A profile is a set of rules that you apply to your registered repositories.
Before creating a profile, you need to ensure that all desired rule_types have been created in Minder.

Start by creating a rule that checks if secret scanning is enabled and creates
a security advisory alert if secret scanning is not enabled.  
This is a reference rule provided by the Minder team in the [minder-rules-and-profiles repository](https://github.com/stacklok/minder-rules-and-profiles).

For this exercise, we're going to download just the `secret_scanning.yaml`
rule, and then use `minder rule_type create` to define the secret scanning rule.

```bash
curl -LO https://raw.githubusercontent.com/stacklok/minder-rules-and-profiles/main/rule-types/github/secret_scanning.yaml

minder rule_type create -f secret_scanning.yaml
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
minder profile_status list --profile github-profile
```
If all registered repositories have secret scanning enabled, you will see the `OVERALL STATUS` is `Success`, otherwise the 
overall status is `Failure`.

See a detailed view of which repositories satisfy the secret scanning rule:
```
minder profile_status list --profile github-profile --detailed
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
If you wish to delete a registered repository, you can do so with the following command:
```bash
minder repo delete -n $REPO_NAME --provider github
```
where `$REPO_NAME` is the fully-qualified name (`owner/name`) of the repository you wish to delete, for example `testorg/testrepo`.
