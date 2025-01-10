---
title: Using Mindev to develop and debug rule types
sidebar_label: Develop and debug rule types
sidebar_position: 120
---

[Mindev](https://github.com/mindersec/minder/tree/main/cmd/dev) is a tool that
helps you develop and debug rule types for Minder. It provides a way to run rule
types locally and test them against your codebase.

While it contains more utilities, this guide focuses on using Mindev to develop
and debug rule types.

## Prerequisites

- [Go](https://golang.org/doc/install) installed on your machine
- [The gh CLI](https://cli.github.com/) installed on your machine

## Build Mindev

```bash
make build-mindev
```

## Run Mindev

```bash
mindev help
```

To see the available options for rule types, run:

```bash
mindev ruletype help
```

## Linting

`ruletype lint` will evaluate the rule, without running it against any external
resources. This will allow you to identify syntax errors quickly. To lint your
rule type, run:

```bash
mindev ruletype lint -r path/to/rule-type.yaml
```

This will give you basic validations on the rule type file.

## Running a rule type

`ruletype test` will execute a rule against an external resource. This will
allow you to test a single rule. You must provide a rule type to evaluate, the
profile to evaluate it in the context of, and the information about the entity
to evaluate.

The entity type must match the rule's `def.in_entity` type; the entity is
defined as a set of YAML properties in the entity file; for example, if you're
testing a rule type that's targetted towards a repository, the YAML must match
the repository schema.

To run a rule type, use the following command:

```bash
mindev ruletype test -e mindev ruletype test -e /path/to/entity -p /path/to/profile -r /path/to/rule
```

Where the flags are:

- `-e` or `--entity`: The path to the entity file
- `-p` or `--profile`: The path to the profile file
- `-r` or `--rule`: The path to the rule file

The entity could be the repository or the codebase you want to test the rule
type against.

The rule is the rule type definition you want to verify

And the profile is needed so we can specify the parameters and definitions for
the rule type.

## Entity

An entity in minder is the target in the supply chain that minder is evaluating.
In some cases, it may be the repository. Minder the minimal information needed
to evaluate the rule type.

The values needed must match an entity's protobuf definition. for instance, for
a repository entity, the following fields are required:

```yaml
---
github/repo_name: <name of the repo>
github/repo_owner: <owner of the repo>
github/repo_id: <upstream ID>
github/clone_url: <clone URL>
github/default_branch: <default branch>
is_private: <true/false>
is_fork: <true/false>
```

Minder is able to use these values to check the current state of the repository
and evaluate the rule type.

You can see examples of the schema for each entity in the
[entity examples](https://github.com/mindersec/minder/tree/main/cmd/dev/examples)
folder.

## Authentication

If the rule type requires authentication, you can use the following environment
variable:

```bash
export TEST_AUTH_TOKEN=your_token
```

You can use [`gh` (the GitHub CLI)](https://github.com/cli/cli) to produce a
GitHub auth token. For example:

```bash
TEST_AUTH_TOKEN=$(gh auth token) mindev ruletype test -e /path/to/entity -p /path/to/profile -r /path/to/rule
```

### Example

Let's evaluate if the `minder` repository has set up dependabot for golang
dependencies correctly.

We can get the necessary rule type from the
[minder rules and profiles repo](https://github.com/mindersec/minder-rules-and-profiles).

We'll create a file called `entity.yaml` with the following content:

```yaml
---
github/repo_name: minder
github/repo_owner: stacklok
github/repo_id: 624056558
github/clone_url: https://github.com/mindersec/minder.git
github/default_branch: main
is_private: false
is_fork: false
```

We'll use the readily available profile for dependabot for golang dependencies:

```yaml
---
# Simple profile showing off the dependabot_configured rule
version: v1
type: profile
name: dependabot-go-github-profile
display_name: Dependabot for Go projects
context:
  provider: github
alert: 'on'
remediate: 'off'
repository:
  - type: dependabot_configured
    def:
      package_ecosystem: gomod
      schedule_interval: daily
      apply_if_file: go.mod
```

This is already available in the
[minder rules and profiles repo](https://github.com/mindersec/minder-rules-and-profiles/blob/main/profiles/github/dependabot_go.yaml).

Let's set up authentication:

```bash
export AUTH_TOKEN=$(gh auth token)
```

Let's give it a try!

```bash
$ mindev ruletype test -e repo.yaml -p profiles/github/dependabot_go.yaml -r rule-types/github/dependabot_configured.yaml
Profile valid according to the JSON schema!
The rule type is valid and the entity conforms to it
```

The output shows that the rule type is valid and the entity conforms to it.
Meaning the `minder` repository has set up dependabot for golang dependencies
correctly.

## Rego print

Mindev also has the necessary pieces set up so you can debug your rego rules.
e.g. `print` statements in rego will be printed to the console.

For more information on the rego print statement, the following blog post is a
good resource:
[Introducing the OPA print function](https://blog.openpolicyagent.org/introducing-the-opa-print-function-809da6a13aee)

## Conclusion

Mindev is a powerful tool that helps you develop and debug rule types for
Minder. It provides a way to run rule types locally and test them against your
codebase.
