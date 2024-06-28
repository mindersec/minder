---
title: Viewing profile status
sidebar_position: 70
---

# Viewing the status of your profile

When you have created a profile and registered repositories, Minder will evaluate your security profile against those repositories. Minder can report on the status, and can optionally alert you using GitHub Security Advisories.

## Prerequisites

Before you can view the status of your profile, you must [create a profile](first_profile).

## Summary profile status

To view the summary status of your profile, use the `minder profile status list` command. If you have a profile named `my_profile`, run:

```bash
minder profile status list --name my_profile
```

If all the registered repositories are in compliance with your profile, you will see the `OVERALL STATUS` column set to `Success`. If one or more repositories are not in compliance, the column will be set to `Failure`.

For example, the profile named `my_profile` expects repositories to have secret scanning enabled. If any repository did not have secret scanning enabled, then the output will look like:

```yaml
+--------------------------------------+------------+----------------+----------------------+
|                  ID                  |    NAME    | OVERALL STATUS |     LAST UPDATED     |
+--------------------------------------+------------+----------------+----------------------+
| 1abcae55-5eb8-4d9e-847c-18e605fbc1cc | my_profile |    Failed      | 2023-11-06T17:42:04Z |
+--------------------------------------+------------+----------------+----------------------+
```

Use detailed status reporting to understand which repositories are not in compliance.

## Detailed profile status

Detailed status will show each repository that is registered, along with the current evaluation status for each rule.

See a detailed view of which repositories satisfy the secret scanning rule:

```bash
minder profile status list --name github-profile --detailed
```

An example output for a profile that checks secret scanning and secret push protection, for an organization that has a single repository registered. In this example, the repository `example/demo_repo` has secret scanning enabled, which is indicated by the `STATUS` column set to `Success`. However, that repository does not have secret push protection enabled, which is indicated by the `STATUS` column set to `Failure`.

```yaml
+--------------------------------------+------------------------+------------+---------+-------------+--------------------------------------+
|               RULE ID                |       RULE NAME        |   ENTITY   | STATUS  | REMEDIATION |             ENTITY INFO              |
+--------------------------------------+------------------------+------------+---------+-------------+--------------------------------------+
| 8a2af1c3-72f6-42ac-a888-45eac5b0f72e | secret_scanning        | repository | Success | Skipped     | provider: github-app-example         |
|                                      |                        |            |         |             | repo_name: demo_repo repo_owner:     |
|                                      |                        |            |         |             | example  repository_id:              |
|                                      |                        |            |         |             | 04055a1a-766e-4f49-a1ba-16ab1e749fef |
|                                      |                        |            |         |             |                                      |
+--------------------------------------+------------------------+------------+---------+-------------+--------------------------------------+
| 08e94b93-e3d6-4df5-a480-ecf108ba481e | secret_push_protection | repository | Failure | Skipped     | provider: github-app-example         |
|                                      |                        |            |         |             | repo_name: demo_repo repo_owner:     |
|                                      |                        |            |         |             | example  repository_id:              |
|                                      |                        |            |         |             | 04055a1a-766e-4f49-a1ba-16ab1e749fef |
|                                      |                        |            |         |             |                                      |
+--------------------------------------+------------------------+------------+---------+-------------+--------------------------------------+
```

## Alerts with GitHub Security Advisories

You can optionally get alerted with [GitHub Security Advisories](https://docs.github.com/en/code-security/security-advisories) when repositories are not in compliance with your security profiles. If you have configured your profile with `alerts: on`, then Minder will generate GitHub Security Advisories.

For example, if you've [created a profile with `alerts: on`](first_profile) that looks for secret scanning to be enabled in your repository, then _disabling_ secret scanning in that repository should produce a GitHub Security Advisory.

In this example, if you [disable secret scanning](https://docs.github.com/en/code-security/secret-scanning/configuring-secret-scanning-for-your-repositories) in one of your registered repositories, Minder will create a GitHub Security Advisory in that repository. To view that, navigate to the repository on GitHub, click on the Security tab and view the Security Advisories. There will be a new advisories named `minder: profile my_profile failed with rule secret_scanning`.

To resolve this issue, you can [enable secret scanning](https://docs.github.com/en/code-security/secret-scanning/configuring-secret-scanning-for-your-repositories) in that repository. When you do this, the advisory will be deleted. If you go back to the Security Advisories page on that repository, you will see that the advisory that was created by Minder has been closed.
