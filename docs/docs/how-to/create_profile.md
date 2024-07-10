---
title: Creating a profile
sidebar_position: 10
---

## Prerequisites

- The `minder` CLI application
- A Minder account with
  [at least `editor` permission](../user_management/user_roles.md)

## Use a reference rule

The first step to creating a profile is to create the rules that your profile
will apply.

The Minder team has provided several reference rules for common use cases. To
get started quickly, create a rule from the set of references.

Fetch all the reference rules by cloning the
[minder-rules-and-profiles repository](https://github.com/stacklok/minder-rules-and-profiles).

```
git clone https://github.com/stacklok/minder-rules-and-profiles.git
```

In that directory you can find all the reference rules and profiles.

```
cd minder-rules-and-profiles
```

Create the `secret_scanning` rule type in Minder:

```
minder ruletype create -f rule-types/github/secret_scanning.yaml
```

## Write your own rule

This section describes how to write your own rule, using the existing rule
`secret_scanning` as a reference. If you've already created the
`secret_scanning` rule, you may choose to skip this section.

Start by creating a rule that checks if secret scanning is enabled.

Create a new file called `secret_scanning.yaml`.

Add some basic information about the rule to the new file, such as the version,
type, name, context, description and guidance.

```yaml
---
version: v1
type: rule-type
name: secret_scanning
context:
  provider: github
description: Verifies that secret scanning is enabled for a given repository.
# guidance is the instructions the user will see if this rule fails
guidance: |
  Secret scanning is a feature that scans repositories for secrets and alerts
  the repository owner when a secret is found. To enable this feature in GitHub,
  you must enable it in the repository settings.

  For more information, see
  https://docs.github.com/en/github/administering-a-repository/about-secret-scanning
```

Next, add the rule definition to the `secret_scanning.yaml` file. Set
`in_entity` to be `repository`, since secret scanning is enabled on the
repository.

```yaml
def:
  in_entity: repository
```

Create a `rule_schema` defining a property describing whether secret scanning is
enabled on a repository.

```yaml
def:
  # ...
  rule_schema:
    properties:
      enabled:
        type: boolean
        default: true
```

Set `ingest` to make a REST call to fetch information about each registered
repository and parse the response as JSON.

```yaml
def:
  # ...
  ingest:
    type: rest
    rest:
      # This is the path to the data source. Given that this will evaluate
      # for each repository in the organization, we use a template that
      # will be evaluated for each repository. The structure to use is the
      # protobuf structure for the entity that is being evaluated.
      endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}"
      parse: json
```

Configure `eval` to use `jq` to read the response from the REST call and
determine if secret scanning is enabled.

```yaml
def:
  # ...
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

Set up the remediation action that will be taken if this rule is not satisfied
(and the profile has turned on remediation). The remediation action in this case
is to make a PATCH request to the repository and enable secret scanning.

```yaml
def:
  # ...
  remediate:
    type: rest
    rest:
      method: PATCH
      endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}"
      body: |
        { "security_and_analysis": {"secret_scanning": { "status": "enabled" } } }
```

Define how users will be alerted if this rule is not satisfied. In this case a
security advisory will be created in any repository that does not satisfy this
rule.

```yaml
def:
  # ...
  alert:
    type: security_advisory
    security_advisory:
      severity: "medium"
```

Putting it all together, you get the following content in
`secret_scanning.yaml`:

```yaml
---
version: v1
type: rule-type
name: secret_scanning
context:
  provider: github
description: Verifies that secret scanning is enabled for a given repository.
guidance: |
  Secret scanning is a feature that scans repositories for secrets and alerts
  the repository owner when a secret is found. To enable this feature in GitHub,
  you must enable it in the repository settings.

  For more information, see
  https://docs.github.com/en/github/administering-a-repository/about-secret-scanning
def:
  in_entity: repository
  rule_schema:
    properties:
      enabled:
        type: boolean
        default: true
  ingest:
    type: rest
    rest:
      endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}"
      parse: json
  eval:
    type: jq
    jq:
      - ingested:
          def: '.security_and_analysis.secret_scanning.status == "enabled"'
        profile:
          def: ".enabled"
  remediate:
    type: rest
    rest:
      method: PATCH
      endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}"
      body: |
        { "security_and_analysis": {"secret_scanning": { "status": "enabled" } } }
  alert:
    type: security_advisory
    security_advisory:
      severity: "medium"
```

Finally, create the `secret_scanning` rule in Minder:

```
minder ruletype create -f secret_scanning.yaml
```

## Create a profile

Now that you've created a secret scanning rule, you can set up a profile that
checks if secret scanning is enabled in all your registered repositories.

Start by creating a file named `profile.yaml`.

Add some basic information about the profile to the new file, such as the
version, type, name and context.

```yaml
version: v1
type: profile
name: my-first-profile
context:
  provider: github
```

Turn on alerting, so that a security advisory will be created for any registered
repository that has not enabled secret scanning.

```yaml
alert: "on"
```

Turn on remediation, so that secret scanning will automatically be enabled for
any registered repositories.

```yaml
remediate: "on"
```

Register the secret scanning rule that you created in the previous step.

```yaml
repository:
  - type: secret_scanning
    def:
      enabled: true
```

Putting it all together, you get the following content if `profile.yaml`:

```yaml
version: v1
type: profile
name: my-first-profile
context:
  provider: github
alert: "on"
remediate: "on"
repository:
  - type: secret_scanning
    name:
      "secret_scanning_github" # Optional, as there aren't multiple rules
      # of the same type under the entity - repository
    def:
      enabled: true
```

Finally, create your profile in Minder:

```bash
minder profile create -f profile.yaml
```

Check the status of your profile and see which repositories satisfy the rules by
running:

```bash
minder profile status list --name my-first-profile --detailed
```

At the moment, the `profile status list` with the `--detailed` flag lists all
the repositories that match the rules. To get a more detailed view of the
profile status, use the `-o json` flag to get the output in JSON format and then
filter the output using `jq`. For example, to get all rules that pertain to the
repository `minder` and have failed, run the following command:

```bash
minder profile status list --name stacklok-remediate-profile -d -ojson 2>/dev/null | jq  -C '.ruleEvaluationStatus | map(select(.entityInfo.repo_name == "minder" and .status == "failure"))'
```

## Defining Rule Names in Profiles

In Minder profiles, rules are identified by their type and, optionally, a unique
name.

### Rule Types vs Rule Names

Rule types are mandatory and refer to the kind of rule being applied. Rule
names, on the other hand, are optional identifiers that become crucial when
multiple rules of the same type exist under an entity.

```yaml
repository:
  - type: secret_scanning
    name: "secret_scanning_github"
    def:
      enabled: true
```

In this example, `secret_scanning` is the rule type and `secret_scanning_github`
is the rule name.

### When are Rule Names Mandatory?

If you're using multiple rules of the same type under an entity, each rule must
have a unique name. This helps distinguish between rules and understand their
specific purpose.

```yaml
repository:
  - type: secret_scanning
    name: "secret_scanning_github"
    def:
      enabled: true
  - type: secret_scanning
    name: "secret_scanning_github_2"
    def:
      enabled: false
```

Here, we have two rules of the same type `secret_scanning` under the
`repository` entity. Each rule has a unique name.

### Uniqueness of Rule Names

No two rules, whether of the same type or different types, can have the same
name under an entity. This avoids confusion and ensures each rule can be
individually managed.

```yaml
repository: # Would return an error while creating
  - type: secret_scanning
    name: "protect_github"
    def:
      enabled: true
  - type: secret_push_protection
    name: "protect_github"
    def:
      enabled: false
```

In the above used example, even though the rules are of different types
(`secret_scanning` and `secret_push_protection`), Minder will return an error
while creating this profile as rule names are same under the same entity. You
may use same rule names under different entities (repository, artifacts, etc.)

Rule name should not match any rule type, except its own rule type. If a rule
name matches its own rule type, it should not conflict with any other rule name
under the same entity, including default rule names. Example:

```yaml
repository: # Would return an error while creating
  - type: dependabot_configured
    name: "dependabot_configured"
    def:
      package_ecosystem: gomod
      schedule_interval: daily
      apply_if_file: go.mod
  - type: dependabot_configured # default 'name' would be 'dependabot_configured'
    def:
      package_ecosystem: npm
      schedule_interval: daily
      apply_if_file: docs/package.json
```

In the above used example, even though the rules names appear different
visually, Minder will return an error while creating this profile as the rule
name for `npm` rule would be `dependabot_configured` internally, which is same
as the explicit name of the `gomod` rule.

### Example

Consider a profile with two `dependabot_configured` rules under the `repository`
entity. The first rule has a unique name, "Dependabot Configured for GoLang".
The second rule doesn't have a name, which is acceptable as Minder would add
rule type as the default name for the rule.

```yaml
repository:
  - type: dependabot_configured
    name: "Dependabot Configured for GoLang"
    def:
      package_ecosystem: gomod
      schedule_interval: daily
      apply_if_file: go.mod
  - type: dependabot_configured # default 'name' would be 'dependabot_configured'
    def:
      package_ecosystem: npm
      schedule_interval: daily
      apply_if_file: docs/package.json
```

You can find the rule definitions used above and many profile examples at
[minder-rules-and-profiles](https://github.com/stacklok/minder-rules-and-profiles)
repository.
