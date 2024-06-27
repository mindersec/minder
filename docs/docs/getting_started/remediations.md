---
title: Automatic Remediations
sidebar_position: 70
---

# Automatic Remediation with Minder

In [Creating your first profile](./first_profile.md), we wrote a rule to open a
security advisory when repo configuration drifted from the configured profile
in Minder.  In this tutorial, we will show how Minder can automatically
resolve the misconfiguration and ensure that enrolled repos have secret
scanning enabled.  Secret scanning is one of several settings which can be
managed by Minder.  When you apply a Minder profile to enrolled repositories,
it will remediate (fix) the setting if it is changed to violate the profile.

## Prerequisites

Before you can enable automatic remediations, you need to [create a security profile](first_profile).

## Creating a profile with `remediate: on`

Minder doesn't currently support editing profiles, so we will create a new profile with `remediate: on`.

Create a new file called `profile-remediate.yaml`.
Paste the following profile definition into the newly created file, setting the `remediate` attribute to `on`:
```yaml
---
version: v1
type: profile
name: github-profile-remediate
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
minder profile create -f profile-remediate.yaml
```

Check the status of the profile:
```
minder profile status list --profile github-profile-remediate
```

With remediation on, the profile status should be "Success" when the repository has been updated to match the profile.
If you navigate to your repository settings with your browser, you should see that the secret scanning
feature is enabled. Toggling the feature off should trigger a new profile status check and the
secret scanning feature should be enabled again in GitHub.

## Current limitations
At the time of writing, not all `rule-type` objects support remediation. To find out which
rule types support remediation, you can run:
```shell
minder ruletype get -i ${ID} -oyaml
```
and look for the `remediate` attribute. If it's not present, the rule type doesn't support
remediation. Alternatively, browse the [rule types directory](https://github.com/stacklok/minder-rules-and-profiles/tree/main/rule-types/github)
of the minder-rules-and-profiles repository.

Furthermore, remediations that open a pull request such as the `dependabot` rule type only attempt
to replace the target file, overwriting its contents. This means that if you want to keep the current
changes, you need to merge the contents manually.
