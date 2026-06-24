---
title: Enrolling the GitLab provider
sidebar_position: 45
---

Once you have authenticated to Minder, you'll need to enroll your GitLab
credentials to allow Minder to manage your GitLab repositories. This allows
Minder to inspect and manage your repository configuration. You will be
prompted to grant Minder access.

This guide uses the OAuth flow, which is the quickest way to get started.
Note that OAuth has some tradeoffs: Minder acts as the authorizing user (which
can make merge requests and attribution confusing), access is hard to scope
when the user belongs to multiple projects, and enrollment breaks if that
user ever leaves the project. For production use with multiple administrators,
a dedicated [service account with a Personal Access Token](../integrations/provider_integrations/gitlab#authorization-methods)
is recommended instead.

## Prerequisites

- [log in to Minder using the CLI](login)

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

