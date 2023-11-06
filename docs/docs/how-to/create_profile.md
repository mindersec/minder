---
title: Creating a profile
sidebar_position: 10
---

## Prerequisites

* The `minder` CLI application
* A Stacklok account

## Use a reference rule
The first step to creating a profile is to create the rules that your profile will apply.

The Minder team has provided several reference rules for common use cases. To get started quickly, create a rule from
the set of references.

Fetch all the reference rules by cloning the [minder-rules-and-profiles repository](https://github.com/stacklok/minder-rules-and-profiles).
```
git clone https://github.com/stacklok/minder-rules-and-profiles.git
```

In that directory you can find all the reference rules and profiles.
```
cd minder-rules-and-profiles
```

Create the `secret_scanning` rule type in Minder:
```
minder rule_type create -f rule-types/github/secret_scanning.yaml
```

## Write your own rule
This section describes how to write your own rule, using the existing rule `secret_scanning` as a reference. If you've
already created the `secret_scanning` rule, you may choose to skip this section.

Start by creating a rule that checks if secret scanning is enabled.  

Create a new file called `secret_scanning.yaml`.

Add some basic information about the rule to the new file, such as the version, type, name, context, description and 
guidance.

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

Next, add the rule definition to the `secret_scanning.yaml` file.
Set `in_entity` to be `repository`, since secret scanning is enabled on the repository.
```yaml
def:
  in_entity: repository
```

Create a `rule_schema` defining a property describing whether secret scanning is enabled on a repository.
```yaml
def:
  # ...
  rule_schema:
      properties:
        enabled:
          type: boolean
          default: true
```

Set `ingest` to make a REST call to fetch information about each registered repository and parse the response as JSON.
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

Configure `eval` to use `jq` to read the response from the REST call and determine if secret scanning is enabled.
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
(and the profile has turned on remediation). The remediation action in this case is to make a PATCH request to the
repository and enable secret scanning.
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

Define how users will be alerted if this rule is not satisfied. In this case a security advisory will be created in
any repository that does not satisfy this rule.
```yaml
def:
  # ...
  alert:
      type: security_advisory
      security_advisory:
        severity: "medium"
```

Putting it all together, you get the following content in `secret_scanning.yaml`:
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
minder rule_type create -f secret_scanning.yaml
```

## Create a profile
Now that you've created a secret scanning rule, you can set up a profile that checks if secret scanning is enabled
in all your registered repositories.

Start by creating a file named `profile.yaml`.

Add some basic information about the profile to the new file, such as the version, type, name and context.
```yaml
version: v1
type: profile
name: my-first-profile
context:
  provider: github
```

Turn on alerting, so that a security advisory will be created for any registered repository that has not enabled
secret scanning.
```yaml
alert: "on"
```

Turn on remediation, so that secret scanning will automatically be enabled for any registered repositories.
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
    def:
      enabled: true
```

Finally, create your profile in Minder:
```
minder profile create -f profile.yaml
```

Check the status of your profile and see which repositories satisfy the rules by running:
```
minder profile_status list --profile my-first-profile --detailed
```
