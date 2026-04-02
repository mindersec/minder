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
[the Minder team on OpenSSF Slack (`#minder`)](https://openssf.slack.com/archives/C07SP9RSM2L)
to get updated status information, or help us deliver that feature by
[contributing to Minder](https://github.com/mindersec/minder/blob/main/CONTRIBUTING.md).

## How to contribute

Have any questions or comments about items on the Minder roadmap? Share your
feedback in Slack or GitHub issues. As they approach implementation, Roadmap
items will start with a tracking design issue, followed by specific
implementation sub-issues, and _may_ use GitHub projects depending on the
complexity.

> Looking for a smaller task?
>
> The Minder repository has a number of
> [good first issues](https://github.com/mindersec/minder/issues?q=state%3Aopen%20label%3A%22good%20first%20issue%22).
> Before starting work on an issue, it's generally a good idea to announce your
> intent on the issue, which can prevent duplicated work if someone else is also
> working on the same item.
>
> Another good place to get started is in the
> [`minder-rules-and-profiles` repository](https://github.com/mindersec/minder-rules-and-profiles/).
> The core goal of Minder is to provide a system for solving supply chain
> security issues, so writing and improving rules and remediations is a good
> place to get started.

**Last updated:** April 2026

## Major Efforts

One key principle behind many of these efforts is that Minder is a supply chain
security _platform_: much of the benefit of a platform is the ability to build
new functionality on top of existing platform primitives, rather than needing to
extend Minder for each new supply chain development. Where possible, the aim is
to enable Minder users to use rule types, profiles, selectors, and other
automation to evolve the platform with minimal changes to the underlying Minder
server.

With that said, there are still a number of core capabilities which have not yet
been built to enable this vision, so this roadmap lays out a number of avenues
for improving Minder's utility and adoption. We will also prioritize features
which improve the user experience for Minder users -- both rule authors
(administrators) and developers working in a project which has adopted Minder.

### Improve Rule Output Handling

**Status**: In progress, tracked in
[#6105](https://github.com/mindersec/minder/issues/6105)

Last year, Minder added the ability to output data from rule evaluations for
consumption by remediations and alerts. We can extend this functionality to
enable a number of additional capabilities with a minimal code footprint.

Some of the suggested capabilities:

- **Inventory rules**: Rather than pass/fail assessment of repository status,
  leverage rule evaluation to collect (for example) all the actions used by a
  repository or all the container images used in all Dockerfiles. Minder rule
  output could then be used to import data into an external software inventory
  control system.

- **Rule-driven child entities**: Leveraging inventory rules, we could add a
  remediation which supports managing (upsert / delete) Minder entities detected
  through an inventory rule. This could even replace some of the built-in
  lifecycle rules (e.g. for PRs).

- **External remediation**: Currently, the Minder server uses a GitHub App with
  high levels of permissions in order to perform remediations. If the data
  needed for remediations (remediation definition, context, and rule output) is
  available through the API, it should be possible to build an external system
  to query rule status and perform remediation using a more powerful token held
  _outside_ of the Minder server. This would reduce the risk / "blast radius" of
  a Minder server compromise, and allow users to adopt hosted Minder scanning
  while keeping powerful GitHub keys under local supervision.

Inventory rules are nearing completion, while the other two bullet points still
need design and discussion.

### Improve Rule Testing Infrastructure

**Status**: Subject of an
[LFX Mentorship Project, Summer 2026](https://mentorship.lfx.linuxfoundation.org/project/40b209ce-c759-4648-9d83-31db4ba1d481)

[Full proposal in Google Docs](https://docs.google.com/document/d/1laRud0GSPqVg_rZ3ahD8GfRJYY2Bf-JL66nEXGZJ4kk/edit).

The primary goals of this 12-week engagement are to:

1. **Improve Rule Testing Tools**: Develop a standalone testing command for
   Minder rules.
1. **Robust Testing**: Allow for the definition and execution of multiple
   distinct test cases against a single rule.
1. **Focus on Local Execution**: Tests should be able to execute locally without
   requiring network access to external resources.
1. **Support Diverse Data Ingestion**: Design the framework to accommodate tests
   for rules covering all current data ingestion methods: REST API, Git
   repositories, and declared data sources.
1. **Provide Automated CI Tooling**: Implement CI workflows to automatically
   detect and run tests.
1. **Update Existing Rules**: Migrate and update existing tests within the
   github.com/mindersec/minder-rules-and-profiles repository to utilize the new
   framework.
1. **Documentation and UX**: Create documentation for the new tooling and
   address potential client-side UX issues related to testing.

### Improve Flexibility of Pull Request Rules

**Status**: Discussion in
[#4452](https://github.com/mindersec/minder/issues/4452), not well-defined yet.

There are several `pull_request` entity evaluators (e.g. OSV, Trusty, Frizbee)
which perform special commenting operations either during rule evaluation or as
a linked action. These include suggesting edits and commenting on specific lines
of content. It would be more flexible to add remediation or alerting action
support for this functionality, which would allow plugging in different
dependency information sources or other tools which can provide line-level
comments.

Functionality under consideration:

- **Line-level comments**: It should be possible to add comments to a PR. This
  should be generally supported across different Git Forges; something like
  [SARIF](https://sarifweb.azurewebsites.net/) (or a reasonable subset) might be
  an appropriate format for recording comments. The line-level comments will
  probably leverage the `output` data described in
  [the previous section](#improve-rule-output-handling).

- **PR checks**: using something like the
  [GitHub Checks API](https://docs.github.com/en/apps/creating-github-apps/writing-code-for-a-github-app/building-ci-checks-with-a-github-app)
  to enable reporting status on specific commits in a PR. This probably requires
  some comparison with the line-level comments approach to determine if both are
  necessary, and how the checks and annotations API differs from general PR
  comments.

- **Enabling content suggestions**: It's not clear whether the GitHub suggestion
  block format is sufficient, or whether Minder should provide a better
  mechanism for suggesting fixes to PRs when the rule remediation supports it
  (for example, pinning GitHub Actions to SHAs during PR review).

Note that CI systems like GitHub Actions can be another route for some
line-level PR checks; we should document when to have Minder directly generate
PR-level comments, and when it makes more sense for Minder to install CI actions
for checking content.

### Expand Provider Coverage

**Status**: Not tracked by a specific issue yet

Minder has _some_ support for [GitLab Cloud](https://gitlab.com/), but setup is
neither well-documented nor well-tested. Additionally, Minder does not currently
support any of the following Git Forges:

- **[Forgejo](https://forgejo.org/)**: not started; this should be designed to
  work with both Codeberg and self-hosted Forgejo, which may require additional
  work on Provider infrastructure
- **GitHub Enterprise**: this work would probably need to be funded
- **GitLab (on-premise)**: again, this might need to be funded

In addition to adding support for Git Forges (where the Minder entity model is
well-tested), Minder has some support for artifact repositories (primarily OCI
repositories such as GHCR and DockerHub). This should be expanded, possibly
following the ["rule-driven child entities"](#improve-rule-output-handling) work
to enable automatic generation of dependent entities given a known parent.

### Human Identity Improvements

**Status**: Under discussion in
[#6217](https://github.com/mindersec/minder/issues/6217)

Minder is currently tightly coupled to Keycloak, but this is a historical
accident, rather than an intentional design. The main requirements for an
identity provider are:

1. Support issuing OpenID Connect (OIDC) JWTs for authentication
2. Support an API for mapping OIDC `sub` claims to user names (for listing
   access to a project)
3. Notification for deleted accounts (to enable GDPR / project cleanup when
   there are no active users)

Note that requirements 2. and 3. are not an intrinsic part of OIDC, and may
require custom code for each provider. Additionally, Minder does not currently
support individual projects registering their own identity providers, which
would allow e.g. a company to grant access to their own governed identities
(human or robot).

### Rule Identity Improvements

**Status**: Not tracked by a specific issue

Minder datasources support
[using the `providerAuth` field](../understand/providers.md#defining-a-data-source)
to authenticate rules to the API which manages the entity, but do not support
any form of other authentication (for example, to a third-party or custom API
endpoint to collect additional data or recommendations). Minder should consider
defining identities for executing rule types (possibly based on the enclosing
project), and using [OIDC](https://openid.net/developers/how-connect-works/) or
[SPIFFE](https://spiffe.io/) to provide a unique identity for each Minder
project or rule executing in the project.

GitHub Actions may
[provide an example](https://docs.github.com/en/actions/reference/security/oidc#oidc-token-claims)
of the data which might be presented alongside a Minder rule identity.

### Status Sharing and Export (Badges)

**Status**: Not tracked by a specific issue

One use for Minder is to evaluate projects against specific conformance
criteria, like the [OpenSSF Security Baseline](https://baseline.openssf.org).
The
[`security-baseline` profile](https://github.com/mindersec/minder-rules-and-profiles/tree/main/security-baseline)
profile enables this tracking, but there is no clear way to share this status
(in comparison with e.g. the [OpenSSF Scorecard](https://scorecard.dev/) and
[Best Practices](https://www.bestpractices.dev/en) badges).

OpenFGA provides an underlying sharing and relationship mechanism for projects
to be able to share _select_ compliance reports either with the world or with
select audiences (such as project sponsors). Currently, Minder does not track
specific resources which could be used to grant "badge-read" or "audit"-type
permissions in OpenFGA. The solution would probably need to support at least:

- Share all reports (policy evaluation results) for an entity.
- Share all reports (policy evaluation results) for an entity and its
  dependents.
- Share one specific report (policy evaluation results) for an entity.

### Improved integration testing

Update [Minder smoke tests](https://github.com/mindersec/smoke-tests), and run a
subset of them automatically against Minder Helm releases, updating / adding
tags to releases which have passed integration tests. This should simplify
running Minder servers and keeping them up to date.

### UI for Minder servers

**Status**: In progress by Custcodian, being considered for donation

Currently private; preview available at https://console.custcodian.dev/

### MCP Server for Minder

**Status**: In progress, proposed for donation by Stacklok

https://github.com/StacklokLabs/minder-mcp

## Future considerations

- **Project hierarchies:** Enable users to create nested projects and group
  repositories within those projects. Projects will inherit profile rules in
  order to simplify profile and policy management.
