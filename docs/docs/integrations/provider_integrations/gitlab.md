---
title: GitLab provider
sidebar_label: GitLab
sidebar_position: 20
---

A provider connects Minder to your software supply chain. It lets Minder know
where to look for your repositories, artifacts, and other entities, in order to
make them available for registration. It also tells Minder how to interact with
your supply chain to enable features such as alerting and remediation. Finally,
it handles the way Minder authenticates to the external service.

## Authorization methods

Minder supports two ways to authorize the GitLab provider: **OAuth** and a
**Personal Access Token (PAT)**. Both grant Minder access to your GitLab
resources, but they differ in how credentials are obtained and managed.

|                     | OAuth                                                                                             | Personal Access Token (PAT)                                             |
| ------------------- | ------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| **Access scope**    | Tied to the authorizing user's GitLab account                                                     | Scoped to a dedicated service account; access can be tightly controlled |
| **Recommended for** | Individual developers and quick evaluations                                                       | Teams and production deployments                                        |
| **How it works**    | Browser-based OAuth 2.0 flow; Minder receives a short-lived token that is refreshed automatically | You generate a token in GitLab and supply it to Minder manually         |
| **Setup effort**    | Low — follow the browser prompt                                                                   | Medium — create a service account, set scopes, copy the token           |
| **Token lifetime**  | Managed automatically by Minder                                                                   | Fixed expiry set at creation; requires manual rotation                  |
| **Rotation**        | Automatic                                                                                         | Manual — re-run `provider enroll` with the existing provider name       |

For production environments, the PAT approach with a dedicated service account
is recommended because it decouples Minder's access from any individual user's
account, survives employee offboarding, and gives you precise control over which
projects Minder can access.

## Prerequisites (PAT method)

Before enrolling GitLab as a provider using a PAT, you will need a Personal
Access Token with sufficient permissions for Minder to read and interact with
your repositories.

### Creating a service account and PAT

It is recommended to create a dedicated **project-linked service account** in
GitLab rather than using a personal account token. This keeps Minder's access
isolated and makes token rotation easier to manage.

1. **Create a service account** by navigating to the **Service accounts** page
   for your group or project and selecting **Add service account**. Enter a name
   and select **Create service account**. See the
   [GitLab service accounts documentation](https://docs.gitlab.com/user/profile/service_accounts/)
   for details on the types of service accounts and any prerequisites (such as
   group Owner or administrator access).

2. **Generate a Personal Access Token** for the service account via the
   **Service accounts** page:
   - Go to the **Service accounts** page.
   - Find the service account, select the vertical ellipsis (**⋮**) next to it,
     and choose **Manage access tokens**.
   - Select **Add new token**.
   - Enter a **Token name** and an **Expiration date** appropriate for your
     organization's rotation policy.
   - Select the following scopes:
     - `api`: required for full API access (repository data, merge requests,
       security findings)
     - `read_repository`: required for reading repository contents
     - `write_repository`: required to create branches to remediate code
       findings via MR.
   - Copy the token value — it will not be shown again.

3. **Add the service account** to the relevant projects or group. Use **Add
   users to a group** or **Add users to a project** in the GitLab UI, or the
   group/project members API.
   [Choose a role](https://docs.gitlab.com/user/permissions/) based on which
   Minder features you need:
   - **Maintainer** _(recommended)_: Grants full access to repository settings,
     branch protection rules, and merge request management. Required if you want
     Minder to apply remediations that modify protected branches or repository
     settings. This is the recommended role for most deployments.
   - **Security Manager** _(minimum)_: Allows Minder to read repository
     contents, security findings, but cannot create merge requests for
     remediations or update most repository settings. Choose this role when you
     want to limit Minder's write access, accepting that most remediation
     actions will not be available.

## Enrolling a provider

### Using a PAT

To enroll GitLab using a Personal Access Token, pass the `--token` flag:

```bash
minder provider enroll --class gitlab --token <your-pat>
```

Once enrolled, your GitLab repositories can be registered with Minder and
security profiles can be applied.

### Using OAuth

To enroll GitLab using the OAuth flow, run:

```bash
minder provider enroll --class gitlab
```

Minder will open a browser window and guide you through the GitLab OAuth
authorization flow. Once you grant access, Minder stores the resulting token and
refreshes it automatically.

## Token rotation

Personal Access Tokens expire according to the expiry date set when they were
created. When a token expires, Minder will no longer be able to communicate with
GitLab and provider operations will fail.

To update Minder with a new token, re-run `provider enroll` using the **existing
provider name** so that Minder updates the credentials in place rather than
creating a duplicate provider:

```bash
minder provider enroll --class gitlab --name <existing-provider-name> --token <new-token>
```

You will be prompted to supply the new PAT. Minder will update the stored
credentials for that provider without affecting any repositories or profiles
already registered under it.

