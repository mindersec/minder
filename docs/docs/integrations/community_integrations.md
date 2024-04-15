---
title: OSS Tooling Integrations
sidebar_position: 30
---

# OSS Tooling Integrations

Minder's policy engine is flexible enough to integrate with a variety of open source tools.
This allows you to leverage the tools you already use to make better decisions about your supply chain.

Most of the integrations supported are done via the policy engine. This is done either as a direct
integration or by using a more dynamic language such as Rego as part of the rule type.

## Trivy

Trivy is a simple and comprehensive vulnerability scanner for repositories and container images.
It can be used to scan your dependencies for known vulnerabilities. Minder integrates with Trivy
by providing a dedicated rule type that ensures that Trivy is configured to run on your repositories.

```bash
$ minder ruletype list
...
+                                      +--------------------------------------+------------------------------------------------------------------------+--------------------------------+
|                                      | XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX | stacklok/trivy_action_enabled                                          | Verifies that the Trivy action |
|                                      |                                      |                                                                        | is enabled for the repository  |
|                                      |                                      |                                                                        | and scanning                   |
+--------------------------------------+--------------------------------------+------------------------------------------------------------------------+--------------------------------+

$ minder ruletype get -i <ruletype_id>
+-------------+---------------------------------------------------------------------------------------+
|  RULE TYPE  |                                        DETAILS                                        |
+-------------+---------------------------------------------------------------------------------------+
| ID          | XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX                                                  |
+-------------+---------------------------------------------------------------------------------------+
| Name        | stacklok/trivy_action_enabled                                                         |
+-------------+---------------------------------------------------------------------------------------+
| Description | Verifies that the Trivy action is enabled for the repository and scanning             |
+-------------+---------------------------------------------------------------------------------------+
| Project     | XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX                                                  |
+-------------+---------------------------------------------------------------------------------------+
| Ingest type | git                                                                                   |
+-------------+---------------------------------------------------------------------------------------+
| Eval type   | rego                                                                                  |
+-------------+---------------------------------------------------------------------------------------+
| Remediation | pull_request                                                                          |
+-------------+---------------------------------------------------------------------------------------+
| Alert       | security_advisory                                                                     |
+-------------+---------------------------------------------------------------------------------------+
| Guidance    | Trivy is an open source vulnerability scanner for repositories, containers and other  |
|             | artifacts provided by Aqua Security. It is used to scan for vulnerabilities in the    |
|             | codebase and dependencies. This rule ensures that the Trivy action is enabled for     |
|             | the repository and scanning is performed.                                             |
|             |                                                                                       |
|             | Set it up by adding the following to your workflow:                                   |
|             |                                                                                       |
|             | ```yaml                                                                               |
|             | - name: Trivy Scan                                                                    |
|             |   uses: aquasecurity/trivy-action@fbd16365eb88e12433951383f5e99bd901fc618f  # v0.12.0 |
|             |   with:                                                                               |
|             |     image-ref: ${{ github.repository }}                                               |
|             |     format: json                                                                      |
|             |     exit-code: 1                                                                      |
|             | ```                                                                                   |
|             |                                                                                       |
|             | For more information, see                                                             |
|             | https://github.com/marketplace/actions/aqua-security-trivy                            |
|             |                                                                                       |
+-------------+---------------------------------------------------------------------------------------+
```

If the rule type is enabled and automatic remediation is configured, Minder will automatically create a pull request to enable Trivy scanning on your repository.

## Dependabot

Dependabot is a tool that helps you keep your dependencies up to date. It automatically creates pull requests to update your dependencies when new versions are available.

Minder integrates with Dependabot by providing a dedicated rule type that ensures that Dependabot is configured to run on your repositories.

```bash
$ minder ruletype list
...
+                                      +--------------------------------------+------------------------------------------------------------------------+--------------------------------+
|                                      | XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX | stacklok/dependabot_configured                                         | Verifies that Dependabot is    |
|                                      |                                      |                                                                        | configured for the repository  |
+                                      +--------------------------------------+------------------------------------------------------------------------+--------------------------------+
...

$ minder ruletype get -i <ruletype_id>
+-------------+----------------------------------------------------------------------------------------------------------------------------------+
|  RULE TYPE  |                                                             DETAILS                                                              |
+-------------+----------------------------------------------------------------------------------------------------------------------------------+
| ID          | XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX                                                                                             |
+-------------+----------------------------------------------------------------------------------------------------------------------------------+
| Name        | stacklok/dependabot_configured                                                                                                   |
+-------------+----------------------------------------------------------------------------------------------------------------------------------+
| Description | Verifies that Dependabot is configured for the repository                                                                        |
+-------------+----------------------------------------------------------------------------------------------------------------------------------+
| Project     | XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX                                                                                             |
+-------------+----------------------------------------------------------------------------------------------------------------------------------+
| Ingest type | git                                                                                                                              |
+-------------+----------------------------------------------------------------------------------------------------------------------------------+
| Eval type   | rego                                                                                                                             |
+-------------+----------------------------------------------------------------------------------------------------------------------------------+
| Remediation | pull_request                                                                                                                     |
+-------------+----------------------------------------------------------------------------------------------------------------------------------+
| Alert       | security_advisory                                                                                                                |
+-------------+----------------------------------------------------------------------------------------------------------------------------------+
| Guidance    | Dependabot enables Automated dependency updates for repositories.                                                                |
|             | It is recommended that repositories have some form of automated dependency updates enabled                                       |
|             | to ensure that vulnerabilities are not introduced into the codebase.                                                             |
|             |                                                                                                                                  |
|             | For more information, see                                                                                                        |
|             | https://docs.github.com/en/code-security/dependabot/dependabot-version-updates/configuration-options-for-the-dependabot.yml-file |
|             |                                                                                                                                  |
+-------------+----------------------------------------------------------------------------------------------------------------------------------+
```

If the rule type is enabled and automatic remediation is configured, Minder will automatically create a pull request to enable Dependabot on your repository.

Note that you need to configure the ecosystem and package managers that dependabot should monitor. This is done by setting up
the relevant parameters in the rule definition in the profile. For example:

```yaml
---
version: v1
type: profile
name: profile-with-dependabot
alert: "on"
remediate: "on"
repository:
  - type: dependabot_configured
    name: go_dependabot
    def:
      package_ecosystem: gomod
      apply_if_file: go.mod
  - type: dependabot_configured
    name: npm_dependabot
    def:
      package_ecosystem: npm
      apply_if_file: package.json
```

In this example, we have two rules that configure Dependabot for Go and NPM packages. The `package_ecosystem` parameter specifies the package manager that
Dependabot should monitor, and the `apply_if_file` parameter specifies the file that should be present in the repository for the rule to apply.

The package ecosystem is anything that dependabot currently supports. For more information, see the
[Dependabot documentation](https://docs.github.com/en/code-security/dependabot/dependabot-version-updates/configuration-options-for-the-dependabot.yml-file).

## OSV (Open Source Vulnerabilities)

OSV is a vulnerability database and triage infrastructure for open source projects. It provides a curated list of vulnerabilities for open source projects
and is used by Minder to check for known vulnerabilities in your dependencies. Minder integrates with OSV by providing a dedicated rule type as well
as a dedicated integration point in the policy engine.

```bash
$ minder ruletype list
...
+                                      +--------------------------------------+------------------------------------------------------------------------+--------------------------------+
|                                      | XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX | stacklok/pr_vulnerability_check                                        | Verifies that pull requests    |
|                                      |                                      |                                                                        | do not add any vulnerable      |
|                                      |                                      |                                                                        | dependencies                   |
+                                      +--------------------------------------+------------------------------------------------------------------------+--------------------------------+
...

$ minder ruletype get -i <ruletype_id>
+-------------+--------------------------------------------------------------------------------------------+
|  RULE TYPE  |                                          DETAILS                                           |
+-------------+--------------------------------------------------------------------------------------------+
| ID          | fedf545d-0143-4bf8-b4f0-ab4548463346                                                       |
+-------------+--------------------------------------------------------------------------------------------+
| Name        | stacklok/pr_vulnerability_check                                                            |
+-------------+--------------------------------------------------------------------------------------------+
| Description | Verifies that pull requests do not add any vulnerable dependencies                         |
+-------------+--------------------------------------------------------------------------------------------+
| Project     | e3d118ab-5dce-4d2d-8772-968b4b9062f4                                                       |
+-------------+--------------------------------------------------------------------------------------------+
| Ingest type | diff                                                                                       |
+-------------+--------------------------------------------------------------------------------------------+
| Eval type   | vulncheck                                                                                  |
+-------------+--------------------------------------------------------------------------------------------+
| Remediation | unsupported                                                                                |
+-------------+--------------------------------------------------------------------------------------------+
| Alert       | security_advisory                                                                          |
+-------------+--------------------------------------------------------------------------------------------+
| Guidance    | For every pull request submitted to a repository, this rule will check if the pull request |
|             | adds a new dependency with known vulnerabilities. If it does, the rule will fail and the   |
|             | pull request will be rejected or commented on.                                             |
|             |                                                                                            |
+-------------+--------------------------------------------------------------------------------------------+
```

As the description and guidance say, this rule type applies to pull requests and checks if any new dependencies added in the pull request have known vulnerabilities.

The rule type is evaluated using the `vulncheck` evaluation type, which is a custom evaluation type that checks the dependencies against the OSV database.
This is a direct minder integration, which means that there is custom code that fetches the vulnerabilities from the OSV database and
checks them against the dependencies in the pull request. It will also comment and propose changes to the pull request if any vulnerabilities are found.

## Conclusion

These were some of the open source tooling integrations that Minder supports. The policy engine is flexible enough to integrate with a variety of tools.
For more custom integrations, contact the Minder team at Stacklok. If you feel adventurous, you can also write your own rule types and integrations.

Here are some resources to get you started:

* https://stacklok.com/blog/how-to-create-new-rule-types-in-minder-to-apply-custom-github-repo-security-settings
* https://stacklok.com/blog/writing-minder-rule-types-with-open-policy-agent-and-rego
