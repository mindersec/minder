---
title: Automatic remediations
sidebar_position: 80
---

# Automatic remediation with Minder

After you've [created a profile](first_profile), you can [view the status](viewing_status) of your security profile, and you can optionally enable alerts through GitHub Security Advisories. But Minder can also automatically remediate your profile, which means that when it detects a repository that is not in compliance with your profile, Minder can attempt to remediate it, bringing it back into compliance.

## Prerequisites

Before you can enable automatic remediations, you need to [add rule types](first_profile#adding-rule-types).

## Enabling automatic remediation

If you added the `secret_scanning` and `secret_push_protection` rules when you [created your first profile](first_profile), then you can update your profile to turn automatic remediation on. When you do this, Minder will identify and repository that doesn't have secret scanning or secret push protection turned on, and will turn it on for you.

## Creating a profile with `remediate: on`

To create a profile that has automatic remediation turned on for `secret_scanning` and `secret_push_protection`, update your `my_profile.yaml`: 

```yaml
---
version: v1
type: profile
name: my_profile
context:
  provider: github
alert: "on"
remediate: "on"
repository:
  - type: secret_scanning
    def:
      enabled: true
  - type: secret_push_protection
    def:
      enabled: true
```

Then update your profile configuration in Minder:

```bash
minder profile apply -f my_profile.yaml
```

## Automatic remediation in action

If you go to the GitHub repository settings for one of your registered repositories, then disable secret scanning, Minder will detect this change and automatically remediate it.

If you reload the page on GitHub, you should see that secret scanning has automatically been re-enabled.

Similarly, when Minder performs an automatic remediation, the profile status should move quickly through two states.

When remediation is on, the profile status should move quickly through three different states. After disabling secret scanning on a repository, check the status of the profile:

```
minder profile status list --name my_profile 
```

1. Immediately, you should see that the profile `STATUS` is set to `Failure`, and the `REMEDIATION` state is set to `Pending`. At this point, Minder will start the automatic remediation.
2. Once Minder has performed the remediation, the profile `STATUS` will remain at `Failure`, but the `REMEDIATION` state will change to `Success`.
3. After Minder has remediated the problem, it will evaluate the rule again. Once this completes, Minder will set the profile `STATUS` to `Success`.

# More information

For more information about automatic remediations, see the [additional documentation in "How Minder works"](../understand/remediations).
