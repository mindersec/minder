---
title: Minder providers
sidebar_position: 10
---

# Providers in Minder

A provider connects Minder to your software supply chain. It lets Minder know where to look for your repositories, in 
order to make them available for registration. It also tells Minder how to interact with your supply change to enable 
features like alerting and remediation.

The currently supported providers are:
* GitHub

Stay tuned as we add more providers in the future!

## Enrolling a provider

To enroll GitHub as a provider, use the following command:
```
minder provider enroll --provider github
```

To enroll a specific GitHub organization, use the following command:
```
minder provider enroll --provider github --owner specific-org
```

Note: If you are enrolling an organization, the account you use to enroll must be an Owner in the organization
or an Admin on the repositories you will be registering.