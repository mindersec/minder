---
title: Quay.io provider
sidebar_label: Quay.io
sidebar_position: 30
---

A provider connects Minder to your software supply chain. It lets Minder know
where to look for your repositories, artifacts, and other entities, in order to
make them available for registration. It also tells Minder how to interact with
your supply chain to enable features such as alerting and remediation. Finally,
it handles the way Minder authenticates to the external service.

## Authorization methods

Minder authorizes the Quay.io provider using a token you supply, similar to
the Personal Access Token (PAT) method for GitLab. Rather than a personal
OAuth token, it is recommended to use a **Robot account** — a non-human,
namespace-scoped credential that Quay.io provides specifically for
automation like Minder. This decouples Minder's access from any individual
user's account, survives employee offboarding, and gives you precise control
over which repositories Minder can read.

## Prerequisites

Before enrolling Quay.io as a provider, you will need a Robot account token
with read access to the repositories you want Minder to monitor.

### Creating and scoping a Robot account

1. **Create a robot account** from the **Robot Accounts** tab of your Quay.io
   organization (or user account, for personal namespaces). Select **Create
   Robot Account** and enter a name — the resulting robot username will be
   `<namespace>+<name>`. See the
   [Quay.io Robot Accounts documentation](https://docs.quay.io/glossary/robot-accounts.html)
   for details.

2. **Set repository permissions** for the robot account: on the robot
   account's **Set repository permissions** page, select the repositories
   you want Minder to access and grant **Read** — Minder only needs to list
   and inspect repositories and images, not push to them.

3. **Copy the robot account's token** from the robot account's detail page.
   This token is what you will supply to Minder during enrollment.

## Enrolling a provider

To enroll Quay.io using a Robot account token, pass the `--token` flag:

```bash
minder provider enroll --class quay --token <robot-account-token>
```

Once enrolled, your Quay.io repositories can be listed by Minder.

## Token rotation

Robot account tokens do not expire automatically, but should be rotated
periodically as part of your organization's credential-hygiene practices, or
immediately if a token is suspected to be compromised. Quay.io lets you
regenerate a robot account's token from the same **Robot Accounts** page
used to create it.

To update Minder with a new token, re-run `provider enroll` using the
**existing provider name** so that Minder updates the credentials in place
rather than creating a duplicate provider:

```bash
minder provider enroll --class quay --name <existing-provider-name> --token <new-token>
```

Minder will update the stored credentials for that provider without
affecting any repositories or profiles already registered under it.
