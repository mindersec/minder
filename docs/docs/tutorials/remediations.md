---
title: Automatic Remediations
sidebar_position: 20
---

## Goal

The goal of this tutorial is to show how can Minder automatically fix
repository settings that do not align with the profile settings and reconcile
the repository with the profile.

## Prerequisites

In order to follow the tutorial, ensure that you have completed the tutorial on
[registering repositories](register_repo_create_profile.md) first.

## Creating a profile with `remediate: on`

At the moment, Minder doesn't support editing profiles. In order to create the
same profile with `remediate: on`, you need to delete the existing profile and create
a new one.

Get the currently installed profiles:
```shell
minder profile list --provider=github
```

Find the ID of the profile you want to remove and delete it:
```shell
minder profile delete -i $ID
```

Edit the YAML file of the profile you want to use and set the `remediate` attribute to
to `on`:
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

Now regardless of the repository settings, the profile status should be "Success".
If you navigate to your repository settings with your browser, you should see that secret scanning
feature is enabled. Toggling the feature off should trigger a new profile status check and the
secret scanning feature should be enabled again in github.

## Current limitations
At the time of writing, not all `rule_type` objects support remediation. To find out which
do, you can run:
```shell
minder rule_type get -i $ID -oyaml
```
and look for the `remediate` attribute. If it's not present, the rule type doesn't support
remediation. Alternatively, browse the [rule types directory](https://github.com/stacklok/minder-rules-and-profiles/tree/main/rule-types/github)
of the minder-rules-and-profiles repository.

Furthermore, remediations that open a pull request such as the `depenabot` rule type only attempt
to replace the target file, overwriting its contents. This means that if you want to keep the current
changes, you need to merge the contents manually.
