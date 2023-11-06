---
title: Automatic Remediations
sidebar_position: 70
---

# Automatic Remediation with Minder

In [Creating your first profile](./first_profile.md), we wrote a rule to open a
security advisory when repo configuration drifted from the configured profile
in Minder.  In this tutorial, we will show how Minder can automatically
resolve the misconfiguration and ensure that enrolled repos have secret
scanning enabled.  Secret scanning isone of several settings which can be
managed by Minder.  When you apply a Minder profile to enrolled repositories,
it will remediate (fix) the setting if it is changed to violate the profile.

## Prerequisites

* [The `minder` CLI application](./install_cli.md)
* [A Minder account](./login.md)
* [An enrolled GitHub token](./login.md#enrolling-the-github-provider) that is either an Owner in the organization or an Admin on the repositories
* [A registered repository in Minder](./first_profile.md#register-repositories)
* [The `secret_scanning`` rule type](./first_profile.md#creating-and-applying-profiles)
* [A policy to open security advisories when secret scanning is off](./first_profile.md#creating-and-applying-profiles)

## Creating a profile with `remediate: on`

Minder doesn't currently support editing profiles, so we will create a new profile with `remediate: on`.

Edit the YAML file of the [profile from the secret-scanning tutorial](./first_profile.md#creating-and-applying-profiles)
and set the `remediate` attribute to `on`:
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

Create the profile in Minder:
```
minder profile create -f profile.yaml
```

Check the status of the profile:
```
minder profile_status list --profile github-profile
```

With remediation on, the profile status should be "Success" when the repository has been updated to match the profile.
If you navigate to your repository settings with your browser, you should see that the secret scanning
feature is enabled. Toggling the feature off should trigger a new profile status check and the
secret scanning feature should be enabled again in GitHub.

## Current limitations
At the time of writing, not all `rule_type` objects support remediation. To find out which
rule types support remediation, you can run:
```shell
minder rule_type get -i $ID -oyaml
```
and look for the `remediate` attribute. If it's not present, the rule type doesn't support
remediation. Alternatively, browse the [rule types directory](https://github.com/stacklok/minder-rules-and-profiles/tree/main/rule-types/github)
of the minder-rules-and-profiles repository.

Furthermore, remediations that open a pull request such as the `dependabot` rule type only attempt
to replace the target file, overwriting its contents. This means that if you want to keep the current
changes, you need to merge the contents manually.
