---
title: Providers
sidebar_position: 20
---

# Providers in Minder

A _provider_ connects Minder to your software supply chain &mdash; giving Minder information about your source code repositories, and their pull requests, dependencies, and artifacts. Minder will apply your [profiles](profiles) to providers to analyze the security posture of your software supply chain, and then will create [alerts](alerts) and can automatically [remediate](remediation) problems that it finds.

The currently supported providers are:
* GitHub

Stay tuned as we add more providers in the future!

## Enrolling a provider

To enroll GitHub as a provider, use the following command:
```
minder provider enroll
```

Note: If you are enrolling an organization, the account you use to enroll must be an Owner in the organization
or an Admin on the repositories you will be registering.

Once a provider is enrolled, public repositories from that provider can be registered with Minder. Security profiles
can then be applied to the registered repositories, giving you an overview of your security posture and providing
remediations to improve your security posture.
