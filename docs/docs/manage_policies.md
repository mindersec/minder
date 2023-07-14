---
id: manage_policies
title: Getting Started (Manage policies and violations)
sidebar_position: 5
slug: /manage_policies
displayed_sidebar: mediator
---

# Getting Started (Manage policies)

In order to detect security deviations from repositories or other entities, Mediator is relying on the concepts of **Policy** and **Violations**.
A policy is a definition of a verification we want to do on an entity in a pipeline. By default, Mediator offers a different set
of **policy types**, covering different aspects of security: repositories, branches, packages, etc...
A **policy** is an instance of a policy type applied to an specific group, with the relevant settings filled in.

When a policy is created for an specific entity, a continuous monitoring for the related objects start. An object can be a repository,
a branch, a package... depending on the policy definition. When an specific object is not matching what's expected,
a **Violation** is raised. When a violation happens, the overall **Policy status** for this specific entity changes,
becoming failed. User can check the reason for this violation and take remediation actions to comply with the policy.

## Prerequisites

- The `medic` CLI application
- A [running mediator instance](./get_started)
- [OAuth Configured](./config_oauth)
- [At least one repository is registered for Mediator](./enroll_user.md)

## List policy types

A policy is associated to a policy type, and a given group ID. Then the policy checks are propagated
against all the repositories belonging to an specific group. A policy type is associated
to an specific provider (currently Github).

Covered policy types are now:

- branch_protection: controls the branch protection rules on a repo
- secret_scanning: enforces secret scanning for a repo

You can list all policy types registered in Mediator:

```bash
medic policy_type list --provider github
```

The format of the policy is being given by the `jsonschema` provided in the policy type.

You can get the schema of a policy type with:

```bash
medic policy_type get --provider github --type branch_protection --schema
```

By default, a policy type is providing some recommended default values, so users can create policies
by using those defaults without having to create a new policy from scratch.

You can get the policy type default values with:

```bash
medic policy_type get --provider github --type branch_protection --default_schema
```

## Create a policy

When there is a need to control the specific behaviours for a set of repositories, a policy can be
created, based on the previous policy types.

A policy needs to be associated with a provider and a group ID, and it will be applied to all
repositories belonging to that group.
The policy can be created by using the provided defaults, or by providing a new one stored on a file.

For creating based on default ones:

```bash
medic policy create --provider github --type branch_protection --default
```

For creating based on a file:

```bash
medic policy create --provider github --type branch_protection --file policy.yaml
```

Where `policy.yaml` has the following format, based on the provided json schema:

```yaml
branches:
  - name: main
    rules:
      required_pull_request_reviews:
        require_code_owner_reviews: true
        required_approving_review_count: 2
      allow_force_pushes: false
      allow_deletions: false
```

When an specific setting is not provided, the value of this setting is not compared against the policy.
This specific policy will monitor the `main` branch for all related repositories, checking that pull request enforcement is on
place, requiring reviews from code owners and a minimum of 2 approvals before landing. It will also require
that force pushes and deletions are disabled for the `main` branch.

When a policy for a provider and group is created, any repos registered for the same provider and group,
are being observed. Each time that there is a change on the repo that causes a policy violation,
the event is triggered and the violation is being captured.

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

3. For an specific repository:

```bash
 medic policy_status list --repo-id 19
```

Where repo-id is the internal ID of the repository. That can be retrieved with:

```bash
medic repo list --provider github --group-id 1
```

Status for a repo can also be retrieved with:

```bash
medic repo get --repo-id 64671386 -g 1 --provider github --status
```

Where `repo-id` is the repo ID for the provider.

## List policy violations

Client can also retrieve the historical of policy violations. A policy violation entry
will inform about:

- policy type
- internal repository ID
- repository owner
- repository name
- metadata: details of the entity that is violated (branch, provider repo id..)
- violation: detailed list of field and expected vs actual value

The historical of policy violations can also be retrieved by client, at different levels:

1. Globally per provider and group, listing all related policy status:

```bash
medic policy_violation list --provider github --group-id 1 --output yaml
```

2. For an specific policy:

```bash
medic policy_status list --policy-id 1
```

or

```bash
medic policy get --id 1 --status --output yaml
```

3. For an specific repository:

```bash
 medic policy_status list --repo-id 19
```

Where repo-id is the internal ID of the repository. That can be retrieved with:

```bash
medic repo list --provider github --group-id 1
```

Status for a repo can also be retrieved with:

```bash
medic repo get --repo-id 64671386 -g 1 --provider github --status
```

Where `repo-id` is the repo ID for the provider.
