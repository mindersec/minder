---
title: Create a GitLab OAuth application
sidebar_position: 75
---

## Prerequisites

- [GitLab](https://gitlab.com) account
- A [local Minder server](run_the_server) running with the `gitlab_provider`
  feature flag enabled (see [Using feature flags](../developer_guide/feature_flags))

## Steps

1. Create a GitLab OAuth application. GitLab supports three ownership levels —
   choose the one that fits your setup:

   - **User-owned:** Go to [User Settings > Applications](https://gitlab.com/-/user_settings/applications)
   - **Group-owned:** Go to your GitLab group → **Settings** → **Applications**
   - **Instance-wide:** Go to **Admin Area** → **Applications** (self-managed only)

2. Click **Add new application** and enter the following details:
   - **Name:** `Minder` (or any name you prefer)
   - **Redirect URI** (add both URIs, one per line):
     ```
     http://localhost:8080/api/v1/auth/callback/gitlab/cli
     http://localhost:8080/api/v1/auth/callback/gitlab/web
     ```
   - **Confidential:** Yes (checked)
   - **Scopes:** Check `api`, `profile`, and `read_repository`

3. Click **Save application**. Copy the **Application ID** and **Secret** — the
   secret is only shown once.

4. Add the following to your `server-config.yaml` under the `provider:` section:

   ```yaml
   provider:
     gitlab:
       client_id: "YOUR_APPLICATION_ID"
       client_secret: "YOUR_SECRET"
       redirect_uri: "http://localhost:8080/api/v1/auth/callback/gitlab"
       webhook_secret: "a-random-secret-string"
       scopes:
         - "api"
         - "profile"
         - "read_repository"
   ```

   The `redirect_uri` should be the base path without `/cli` or `/web` — Minder
   appends the correct suffix automatically.

5. Enable the `gitlab_provider` feature flag by creating `flags-config.yaml` in
   the root of your Minder directory:

   ```yaml
   gitlab_provider:
     variations:
       enabled: true
       disabled: false
     defaultRule:
       variation: enabled
   ```

6. (Re)start the Minder server:

   ```bash
   make run-docker
   ```

7. Enroll the GitLab provider using the CLI:

   ```bash
   minder provider enroll --class gitlab
   ```

   A browser window will open to GitLab's OAuth authorization page. After
   authorizing, the browser will show **Minder enrollment complete** and the CLI
   will print `Provider enrolled successfully`.

## Access model

Minder acts as the authenticated GitLab user when managing repositories. This
means:

- If the enrolling user loses access to a repository (e.g. leaves a project or
  organization), Minder will no longer be able to enforce policy on that
  repository.
- To restore access, re-enroll the provider with a user who has access:
  `minder provider enroll --class gitlab`

For production use, consider using a dedicated service account or group-owned
OAuth application to avoid disruption if individual team members leave.

## Known limitations

- GitLab support is currently only available on self-hosted Minder instances.
  The hosted instance at `api.custcodian.dev` does not yet support GitLab
  enrollment.
- Webhook-based event delivery requires an externally reachable URL. For local
  development, tools like [ngrok](https://ngrok.com) can expose your local
  server.
- PR remediation (auto-creating branches/PRs) is not yet implemented for GitLab.
- Container registry and artifact support is not yet implemented for GitLab.
- GitLab service account PATs are not currently supported due to a validation
  issue with the `.` character in PAT tokens.
- Token identity verification after enrollment is not yet implemented.
