---
title: Enrolling the GitHub Provider
sidebar_position: 40
---

# Enrolling the GitHub Provider

Once you have authenticated to Minder, you'll need to enroll your GitHub credentials to allow Minder to manage your GitHub repositories. This allows Minder to inspect and manage your repository configuration. You will be prompted to grant Minder access.

In the future, Minder will support other source control and artifact repositories, and you will be able to enroll credentials for those providers in the same manner.

:::note
If you used the [minder `quickstart` command](quickstart), the GitHub Provider was enrolled as part of the quickstart, and you do not need to enroll a second time.
:::

## Prerequisites

Before you can enroll the GitHub Provider, you must [log in to Minder using the CLI](login).

## Enrolling and granting access

To enroll your GitHub credentials in your Minder account, run:

```bash
minder provider enroll
```

A browser session will open, and you will be prompted to login to your authorize the Minder application to access your GitHub account and select the organizations you want to install the application for.

![Enrollment screenshot](./minder-authorize.png)
![Enrollment screenshot](./minder-enroll.png)

Once you have granted Minder access, you will be redirected back, and the user will be enrolled. The `minder` CLI application will report the session is complete.
