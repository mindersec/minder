---
title: Enrolling a Provider
sidebar_position: 40
---

# Enrolling a Provider

Once you have authenticated to Minder, you'll want to enroll your GitHub credentials to allow Minder to manage your GitHub repositories.  In the future, Minder will support other source control and artifact repositories, and you will be able to enroll credentials for those providers in the same manner.

## Prerequisites

* A running Minder server, including a running KeyCloak installation
* A GitHub account
* [The `minder` CLI application](./install_cli.md)
* [Logged in to Minder server](./login.md)

## Enrolling the GitHub Provider

To enroll your GitHub credentials in your Minder account, run:

```bash
minder provider enroll
```

A browser session will open, and you will be prompted to login to your GitHub account.

![Enrollment screenshot](./enroll-screenshot.png)

Once you have granted Minder access, you will be redirected back, and the user will be enrolled. The `minder` CLI application will report the session is complete.

When enrolling an organization, use the `--owner` flag of the `minder provider enroll` command to specify the organization name:
```bash
minder provider enroll --owner test-org
```
The `--owner` flag is not required when enrolling repositories from your personal account.

Note: If you are enrolling an organization, the account you use to enroll must be an Owner in the organization
or an Admin on the repositories you will be registering.
