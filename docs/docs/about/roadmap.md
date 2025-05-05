---
title: Roadmap
sidebar_position: 70
---

## About this roadmap

This roadmap should serve as a reference point for Minder users and community
members to understand where the project is heading. The roadmap is where you can
learn about what features we're working on, what stage they're in, and when we
expect to bring them to you. Priorities and requirements may change based on
community feedback, roadblocks encountered, community contributions, and other
factors. If you depend on a specific item, we encourage you to reach out to
[the Minder team on OpenSSF Slack
(`#minder`)](https://openssf.slack.com/archives/C07SP9RSM2L) to get updated
status information, or help us deliver that feature by contributing to Minder.

## How to contribute

Have any questions or comments about items on the Minder roadmap? Share your
feedback via
[Minder GitHub Discussions](https://github.com/mindersec/minder/discussions).

**Last updated:** June 2024

## In progress

- **Improved information about alerts:** Improve the verbiage and explanation
  about the state of rule evaluations, and how you can remediate the problems.
- **Enforce license information for dependencies:** Ensure that dependencies in
  your repositories use licenses that you approve.
- **Create policy to manage licenses in PRs:** Add a rule type to block and/or
  add comments to pull requests based on the licenses of the dependencies they
  import.
- **Generalized "provider" support:** Improve the ability for developers to add
  integration points to Minder to provide custom information about entities in
  their software development lifecycle.

## Next

- **Report CVEs and license info for ingested SBOMs:**
  Ingest SBOMS and identify dependencies; show CVEs, and license information
  including any changes over time.
- **Block PRs based on Minder rules:** In addition to adding comments to pull
  requests (as is currently available), add the option to block pull requests
  as a policy remediation.
- **Policy events:** Provide information about rule evaluation as it changes,
  and historical rule evaluation.
- **Generate SBOMs:** Enable users to automatically create and sign SBOMs.

## Future considerations

- **Project hierarchies:** Enable users to create nested projects and group
  repositories within those projects. Projects will inherit profile rules in
  order to simplify profile and policy management.
- **Automate the generation and signing of SLSA provenance statements:** Enable
  users to generate SLSA provenance statements (e.g. through SLSA GitHub
  generator) and sign them with Sigstore.
- **Register GitLab and Bitbucket repositories:** In addition to managing GitHub
  repositories, enable users to manage configuration and policy for other source
  control providers.
- **Export a Minder 'badge/certification' that shows what practices a project
  followed:** Create a badge that OSS maintainers and enterprise developers can
  create and share with others that asserts the Minder practices and policies
  their projects follow.
- **Temporary permissions to providers vs. long-running:** Policy remediation
  currently requires long-running permissions to providers such as GitHub;
  provide the option to enable temporary permissions.
- **Create PRs for dependency updates:** As a policy autoremediation option,
  enable Minder to automatically create pull requests to update dependencies
  based on vulnerabilities or license changes.
- **Drive policy through git (config management):** Enable users to dynamically
  create and maintain policies from other sources, e.g. Git, allowing for easier
  policy maintenance and the ability to manage policies through GitOps
  workflows.
- **Integrations with additional OSS and commercial tools:** Integrate with
  tools that run code and secrets scanning (eg Snyk), and behavior analysis (eg
  [OSSF Package Analysis tool](https://github.com/ossf/package-analysis)).
- **Help package authors improve OpenSSF Scorecard scores:** Provide policies
  to improve OpenSSF Scorecard scores through targeted remediations.
