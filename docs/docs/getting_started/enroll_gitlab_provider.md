---
title: Enrolling the GitLab provider
sidebar_position: 45
---

Once you have authenticated to Minder, you'll need to enroll your GitLab
credentials to allow Minder to manage your GitLab repositories. This allows
Minder to inspect and manage your repository configuration. You will be
prompted to grant Minder access.

## Prerequisites

Before you can enroll the GitLab provider, you must:

- [log in to Minder using the CLI](login)
- have a [GitLab OAuth application configured](../run_minder_server/config_gitlab_oauth)
  on your Minder server

## Enrolling and granting access

To enroll your GitLab credentials in your Minder account, run:

```bash
minder provider enroll --class gitlab
```

A browser session will open, directing you to GitLab's authorization page.
Unlike GitHub, there is no organization or group selection step — GitLab
grants access based on the scopes configured in the OAuth application
(`api`, `read_user`, and `read_repository`).

Once you authorize Minder within GitLab, the browser window will close, and
the `minder` CLI will report:

```bash
Provider enrolled successfully
```

## More information

Once enrolled, you can
[register your GitLab repositories](register_repos_gitlab) with Minder.

