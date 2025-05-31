---
title: Managing Minder With GitHub Actions
sidebar_position: 60
---

In addition to [human users](./adding_users.md), Minder also supports
authenticating GitHub Actions.  GitHub Actions are [identified by the `sub`
claim on the GitHub-issued JWT](https://docs.github.com/en/actions/security-for-github-actions/security-hardening-your-deployments/about-security-hardening-with-openid-connect#example-subject-claims).
Unlike human users, GitHub Actions _cannot_ accept an invitation, so
permissions must be assigned directly with `project role grant` or the
[PermissionsService.AssignRole](https://mindersec.github.io/ref/proto#minder-v1-PermissionsService)
API.  Unlike human users, GitHub Actions are identified using the format

```
githubactions/${sub}
```

For example, a GitHub Action run from the `main` branch of the `.github`
repository in the `example-org` organization (a common configuration) would be
specified as `githubactions/repo:example-org/.github:ref:refs/heads/main`.  You
could grant this role `admin` permission on a project with the following
command:

```
minder project role grant --grpc-host api.custcodian.dev \
  --project 00000000-0000-0000-0000-000000000000 \
  --sub githubactions/repo:myorg/myrepo:ref:refs/heads/main \
  --role admin
```

You can then use a GitHub action like [the Custcodian minder
action](https://github.com/custcodian/minder-action) to load rule types and
profiles from your `.github` repository.

## Configuring Minder For GitHub Actions Authentication

As a Minder administrator, there are two settings which need to be enabled
to allow GitHub Actions to authenticate.  The first is to add the GitHub
Actions OIDC issuer to the `identity` configuration section of the Minder
server:

```yaml
identity:
  server:
    # ...
  additional_issuers:
  - https://token.actions.githubusercontent.com
```

The second step is to enable the `machine_accounts` experiment:

```yaml
machine_accounts:
  variations:
    enabled: true
    disabled: false
  defaultRule:
    variation: enabled
```