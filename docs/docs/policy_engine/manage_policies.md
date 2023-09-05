---
id: manage_policies
title: Manage policies and violations
sidebar_position: 2
slug: /manage_policies
displayed_sidebar: mediator
---

# Manage policies

In order to detect security deviations from repositories or other entities, Mediator is relying on the concepts of **Policies**.
A policy is a definition of a verification we want to do on an entity in a pipeline.
A **policy** is an instance of a policy type applied to an specific group, with the relevant settings filled in.

An example policy is the following:

```yaml
---
version: v1
type: pipeline-policy
name: acme-github-policy
context:
  organization: ACME
  group: Root Group
repository:
  - context: github
    rules:
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

The full example is available in the [examples directory](https://github.com/stacklok/mediator/blob/main/examples/github/policies/policy.yaml).

This policy is checking that secret scanning is enabled for all repositories belonging to the ACME organization,
and that the `main` branch is protected, requiring at least one approval from a code owner before landing a pull request.

You'll notice that this policy calls two different rules: `secret_scanning` and `branch_protection`.

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
  group: Root Group
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
      endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Repository}}"
      # This is the method to use to retrieve the data. It should already default to JSON
      parse: json
  # Defines the configuration for evaluating data ingested against the given policy
  eval:
    type: jq
    jq:
      # Ingested points to the data retrieved in the `ingest` section
      - ingested:
          def: '.security_and_analysis.secret_scanning.status == "enabled"'
        # policy points to the policy itself.
        policy:
          def: ".enabled"

```

The full example is available in the [examples directory](https://github.com/stacklok/mediator/tree/main/examples/github/rule-types)

This rule type is checking that secret scanning is enabled for all repositories belonging to the ACME organization.

The rule type defines how the upstream GitHub API is to be queried, and how the data is to be evaluated.
It also defines how instances of this rule will be validated against the rule schema.

When a policy is created for an specific group, a continuous monitoring for the related objects start. An object can be a repository,
a branch, a package... depending on the policy definition. When an specific object is not matching what's expected,
a violation is presented via the policy's **status**. When a violation happens, the overall **Policy status** for this specific entity changes,
becoming failed. There is also individual statuses for each rule evaluation. User can check the reason for this violation and take remediation
actions to comply with the policy.

## Prerequisites

- The `medic` CLI application
- A [running mediator instance](./getting_started)
- [OAuth Configured](./config_oauth)
- [At least one repository is registered for Mediator](./enroll_user.md)

## List rule types

Covered rule types are now:

- branch_protection: controls the branch protection rules on a repo
- secret_scanning: enforces secret scanning for a repo

You can list all policy types registered in Mediator:

```bash
medic rule_type list --provider github
```

By default, a rule type is providing some recommended default values, so users can create policies
by using those defaults without having to create a new policy from scratch.

## Create a rule type

Before creating a policy, we need to ensure that all rule types exist in mediator.

A rule type can be created by pointing to a file containing the rule type definition:

```bash
medic rule_type create -f ./examples/github/rule-types/secret_scanning.yaml
```

Where `secret_scanning.yaml` may look as the example above.

Once all the relevant rule types are available for our group, we may take them into use
by creating a policy.

## Create a policy

When there is a need to control the specific behaviours for a set of repositories, a policy can be
created, based on the previous policy types.

A policy needs to be associated with a provider and a group ID, and it will be applied to all
repositories belonging to that group.
The policy can be created by using the provided defaults, or by providing a new one stored on a file.

For creating based on a file:

```bash
medic policy create --provider github -f ./examples/github/policies/policy.yaml
```

Where `policy.yaml` may look as the example above.

When an specific setting is not provided, the value of this setting is not compared against the policy.
This specific policy will monitor the `main` branch for all related repositories, checking that pull request enforcement is on
place, requiring reviews from code owners and a minimum of 2 approvals before landing. It will also require
that force pushes and deletions are disabled for the `main` branch.

When a policy for a provider and group is created, any repos registered for the same provider and group,
are being observed. Each time that there is a change on the repo that causes the policy status to be updated.

## List policy status

When there is an event that causes a policy violation, the violation is stored in the database, and the
overall status of the policy for this specific repository is changed.
Policy status will inform about:

- policy_type (branch_protection...)
- status: [success, failure]
- last updated: time when this status was updated

Policy status can be checked at different levels:

1. Globally per provider and group, listing all related policy status:

```bash
medic policy_status list --provider github --group-id 1
```

2. For an specific policy:

```bash
medic policy_status list --policy-id 1
```

or

```bash
medic policy get --id 1 --status --output yaml
```
