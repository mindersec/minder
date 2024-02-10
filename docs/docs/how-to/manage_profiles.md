---
title: Manage profiles and violations
sidebar_position: 20
---

# Manage profiles

In order to detect security deviations from repositories or other entities, Minder is relying on the concepts of **Profiles**.
A profile is a definition of a verification we want to do on an entity in a pipeline.
A **profile** is an instance of a profile type applied to an specific group, with the relevant settings filled in.

An example profile is the following:

```yaml
---
version: v1
type: profile
name: acme-github-profile
context:
  provider: github
repository:
  - type: secret_scanning
    def:
      enabled: true
  - type: branch_protection
    params:
      branch: main
    def:
      required_pull_request_reviews:
        dismiss_stale_reviews: true
        require_code_owner_reviews: true
        required_approving_review_count: 1
      required_linear_history: true
      allow_force_pushes: false
      allow_deletions: false
      allow_fork_syncing: true
```

The full example is available in the [examples directory](https://github.com/stacklok/minder-rules-and-profiles).

This profile is checking that secret scanning is enabled for all repositories and that the `main` branch is protected, 
requiring at least one approval from a code owner before landing a pull request.

You'll notice that this profile calls two different rules: `secret_scanning` and `branch_protection`.

Rules can be instantiated from rule types, and they are the ones that are actually doing the verification.

A rule type is a definition of a verification we want to do on an entity in a pipeline.

An example rule type is the following:

```yaml
---
version: v1
type: rule-type
name: secret_scanning
context:
  provider: github
description: Verifies that secret scanning is enabled for a given repository.
def:
  # Defines the section of the pipeline the rule will appear in.
  # This will affect the template that is used to render multiple parts
  # of the rule.
  in_entity: repository
  # Defines the schema for writing a rule with this rule being checked
  rule_schema:
    properties:
      enabled:
        type: boolean
        default: true
  # Defines the configuration for ingesting data relevant for the rule
  ingest:
    type: rest
    rest:
      # This is the path to the data source. Given that this will evaluate
      # for each repository in the organization, we use a template that
      # will be evaluated for each repository. The structure to use is the
      # protobuf structure for the entity that is being evaluated.
      endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}"
      # This is the method to use to retrieve the data. It should already default to JSON
      parse: json
  # Defines the configuration for evaluating data ingested against the given profile
  eval:
    type: jq
    jq:
      # Ingested points to the data retrieved in the `ingest` section
      - ingested:
          def: '.security_and_analysis.secret_scanning.status == "enabled"'
        # profile points to the profile itself.
        profile:
          def: ".enabled"

```

The full example is available in the [examples directory](https://github.com/stacklok/minder-rules-and-profiles)

This rule type is checking that secret scanning is enabled for all repositories.

The rule type defines how the upstream GitHub API is to be queried, and how the data is to be evaluated.
It also defines how instances of this rule will be validated against the rule schema.

When a profile is created for an specific group, a continuous monitoring for the related objects start. An object can be a repository,
a branch, a package... depending on the profile definition. When an specific object is not matching what's expected,
a violation is presented via the profile's **status**. When a violation happens, the overall **Profile status** for this specific entity changes,
becoming failed. There is also individual statuses for each rule evaluation. User can check the reason for this violation and take remediation
actions to comply with the profile.

## Prerequisites

- The `minder` CLI application
- [At least one repository is registered for Minder](../getting_started/register_repos.md)

## List rule types

Covered rule types are now:

- branch_protection: controls the branch protection rules on a repo
- secret_scanning: enforces secret scanning for a repo

You can list all profile types registered in Minder:

```bash
minder ruletype list
```

By default, a rule type is providing some recommended default values, so users can create profiles
by using those defaults without having to create a new profile from scratch.

## Create a rule type

Before creating a profile, we need to ensure that all rule types exist in Minder.

A rule type can be created by pointing to a directory (or file) containing the rule type definition:

```bash
minder ruletype create -f ./examples/github/rule-types
```

Where the yaml files in the directory `rule-types` may look as the example above.

Once all the relevant rule types are available for our group, we may take them into use
by creating a profile.

## Create a profile

When there is a need to control the specific behaviours for a set of repositories, a profile can be
created, based on the previous profile types.

A profile needs to be associated with a provider and a group ID, and it will be applied to all
repositories belonging to that group.
The profile can be created by using the provided defaults, or by providing a new one stored on a file.

For creating based on a file:

```bash
minder profile create -f ./examples/github/profiles/profile.yaml
```

Where `profile.yaml` may look as the example above.

When an specific setting is not provided, the value of this setting is not compared against the profile.
This specific profile will monitor the `main` branch for all related repositories, checking that pull request enforcement is on
place, requiring reviews from code owners and a minimum of 2 approvals before landing. It will also require
that force pushes and deletions are disabled for the `main` branch.

When a profile for a provider and group is created, any repos registered for the same provider and group,
are being observed. Each time that there is a change on the repo that causes the profile status to be updated.

## List profile status

When there is an event that causes a profile violation, the violation is stored in the database, and the
overall status of the profile for this specific repository is changed.
Profile status will inform about:

- profile_type (branch_protection...)
- status: [success, failure]
- last updated: time when this status was updated

Profile status can be checked using the following commands

```bash
minder profile status list --name github-profile
```

To view all of the rule evaluations, use the following

```bash
minder profile status list --name github-profile --detailed
```
