---
sidebar_position: 120
---

# Using Mindev to develop and debug rule types

[Mindev](https://github.com/stacklok/minder/tree/main/cmd/dev) is a tool that helps you develop and debug rule types for Minder. It provides a way to run rule types locally and test them against your codebase.

While it contains more utilities, this guide focuses on using Mindev to develop and debug rule types.

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

To lint your rule type, run:

```bash
mindev ruletype lint -f path/to/rule-type.yaml
```

## Running a rule type

To run a rule type, use the following command:

```bash
mindev ruletype test -e mindev ruletype test -e /path/to/entity -p /path/to/profile -r /path/to/rule
```

Where the flags are:

- `-e` or `--entity`: The path to the entity file
- `-p` or `--profile`: The path to the profile file
- `-r` or `--rule`: The path to the rule file

The entity could be the repository or the codebase you want to test the rule type against.

The rule is the rule type definition you want to verify

And the profile is needed so we can specify the parameters and definitions for the rule type.

## Entity

An entity in minder is the target in the supply chain that minder is evaluating. In some cases, it may
be the repository. Minder the minimal information needed to evaluate the rule type.

The values needed must match an entity's protobuf definition. for instance, for a repository entity, the following fields are required:

```yaml
---
name: <name of the repo>
owner: <owner of the repo>
repo_id: <upstream ID>
clone_url: <clone URL>
default_branch: <default branch>
```

Minder is able to use these values to check the current state of the repository and evaluate the rule type.

## Authentication

If the rule type requires authentication, you can use the following environment variable:

```bash
export AUTH_TOKEN=your_token
```

### Example

Let's evaluate if the `minder` repository has set up dependabot for golang dependencies correctly.

We can get the necessary rule type from the [minder rules and profiles repo](https://github.com/stacklok/minder-rules-and-profiles).

We'll create a file called `entity.yaml` with the following content:

```yaml
---
name: minder
owner: stacklok
repo_id: 624056558
clone_url: https://github.com/stacklok/minder.git
default_branch: main
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
alert: "on"
remediate: "off"
repository:
  - type: dependabot_configured
    def:
      package_ecosystem: gomod
      schedule_interval: daily
      apply_if_file: go.mod
```

This is already available in the [minder rules and profiles repo](https://github.com/stacklok/minder-rules-and-profiles/blob/main/profiles/github/dependabot_go.yaml).

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

The output shows that the rule type is valid and the entity conforms to it. Meaning the `minder` repository has set up dependabot for golang dependencies correctly.

## Rego print

Mindev also has the necessary pieces set up so you can debug your rego rules. e.g. `print` statements
in rego will be printed to the console.

For more information on the rego print statement, the following blog post is a good resource: [Introducing the OPA print function](https://blog.openpolicyagent.org/introducing-the-opa-print-function-809da6a13aee)

## Conclusion

Mindev is a powerful tool that helps you develop and debug rule types for Minder. It provides a way to run rule types locally and test them against your codebase.
