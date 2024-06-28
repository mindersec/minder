---
sidebar_position: 110
---

# Writing rules using Rego 

Minder's policy engine is able to use pluggable drivers for evaluating rules. Rego is a language specifically designed for expressing policies in a clear and concise manner. Its declarative syntax makes it an excellent choice for defining policy logic. In the context of Minder, Rego plays a central role in crafting Rule Types, which are used to enforce security policies.

# Writing Rule Types in Minder

Minder organizes policies into Rule Types, each with specific sections defining how policies are ingested, evaluated, and acted upon. Rule types are then called within profiles to express the security posture of your organization. Let's delve into the essential components of a Minder Rule Type:

* Ingesting Data: Fetching relevant data, often from external sources like GitHub API.

* Evaluation: Applying policy logic to the ingested data. Minder offers a set of engines to evaluate data: jq and rego being general-purpose engines, while trusty and vulncheck are more use case-specific ones.

* Remediation and Alerting: Taking actions or providing notifications based on evaluation results. E.g. creating a pull request or generating a GitHub security advisory.

## Rego Evaluation types

With Rego being a flexible policy language, it allowed us to express policy checks via different constructs. We chose to implement two in Minder:

* **deny-by-default**: Checks for an allowed boolean being set to true, and denies the policy if it’s not the case.

* **constraints**: Checks for violations in the given policy. This allows us to express the violations and output them in a user friendly-manner.

Note that these are known patterns in the OPA community, so we’re not doing anything out of the ordinary here. Instead, we leverage best practices that have already been established.

## Custom Rego functions

Given the context in which Minder operates, we did need to add some custom functionality that OPA doesn’t provide out of the box. Namely, we added the following custom functions:

* **file.exists**: Verifies that the given file exists in the Git repository.

* **file.read**: Reads the contents of the given file in the Git repository.

* **file.ls**: Lists files in the given directory in the Git repository.

* **file.ls_glob**: Lists files in the given directory in the Git repository that match the given glob pattern.

* **file.http_type**: Returns the HTTP content type of the given file.

* **file.walk**: Walks the given directory in the Git repository and lists all files.

* **github_workflow.ls_actions**: Lists all actions in the given GitHub workflow directory.

## Example: CodeQL-Enabled Check

CodeQL is a very handy tool that GitHub provides to do static analysis on codebases. In this scenario, we’ll see a rule type that verifies that it’s enabled via a GitHub action in the repository.

```yaml
---
version: v1
type: rule-type
name: codeql_enabled
context:
  provider: github
description: Verifies that CodeQL is enabled for the repository
guidance: |
  CodeQL is a tool that can be used to analyze code for security vulnerabilities.
  It is recommended that repositories have some form of static analysis enabled
  to ensure that vulnerabilities are not introduced into the codebase.

  To enable CodeQL, add a GitHub workflow to the repository that runs the
  CodeQL analysis.

  For more information, see
  https://docs.github.com/en/code-security/secure-coding/automatically-scanning-your-code-for-vulnerabilities-and-errors/configuring-code-scanning#configuring-code-scanning-for-a-private-repository
def:
  # Defines the section of the pipeline the rule will appear in.
  # This will affect the template used to render multiple parts
  # of the rule.
  in_entity: repository
  # Defines the schema for writing a rule with this rule being checked
  rule_schema:
    type: object
    properties:
      languages:
        type: array
        items:
          type: string
        description: |
          Only applicable for remediation. Sets the CodeQL languages to use in the workflow.
          CodeQL supports 'c-cpp', 'csharp', 'go', 'java-kotlin', 'javascript-typescript', 'python', 'ruby', 'swift'
      schedule_interval:
        type: string
        description: |
          Only applicable for remediation. Sets the schedule interval for the workflow.
    required:
      - languages
      - schedule_interval
  # Defines the configuration for ingesting data relevant for the rule
  ingest:
    type: git
    git:
      branch: main
  # Defines the configuration for evaluating data ingested against the given profile
  eval:
    type: rego
    rego:
      type: deny-by-default
      def: |
        package minder

        default allow := false

        allow {
            # List all workflows
            workflows := file.ls("./.github/workflows")

            # Read all workflows
            some w
            workflowstr := file.read(workflows[w])

            workflow := yaml.unmarshal(workflowstr)

            # Ensure a workflow contains the codel-ql action
            some i
            steps := workflow.jobs.analyze.steps[i]
            startswith(steps.uses, "github/codeql-action/analyze@")
        }
  # Defines the configuration for alerting on the rule
  alert:
    type: security_advisory
    security_advisory:
      severity: "medium"
```

The rego evaluation uses the `deny-by-default` type. It’ll set the policy as successful if there is a GitHub workflow that instantiates `github/codeql-action/analyze`.

## Example: No 'latest' tag in Dockerfile

In this scenario, we’ll explore a Rule Type that verifies that a Dockerfile does not use the `latest` tag.

```yaml
---
version: v1
type: rule-type
name: dockerfile_no_latest_tag
context:
  provider: github
description: Verifies that the Dockerfile image references don't use the latest tag
guidance: |
  Using the latest tag for Docker images is not recommended as it can lead to unexpected behavior.
  It is recommended to use a checksum instead, as that's immutable and will always point to the same image.
def:
  # Defines the section of the pipeline the rule will appear in.
  # This will affect the template used to render multiple parts
  # of the rule.
  in_entity: repository
  # Defines the schema for writing a rule with this rule being checked
  # In this case there are no settings that need to be configured
  rule_schema: {}
  # Defines the configuration for ingesting data relevant for the rule
  ingest:
    type: git
    git:
      branch: main
  # Defines the configuration for evaluating data ingested against the given profile
  # This example verifies that image in the Dockerfile do not use the 'latest' tag
  # For example, this will fail:
  # FROM golang:latest
  # These will pass:
  # FROM golang:1.21.4
  # FROM golang@sha256:337543447173c2238c78d4851456760dcc57c1dfa8c3bcd94cbee8b0f7b32ad0
  eval:
    type: rego
    rego:
      type: constraints
      def: |
        package minder

        violations[{"msg": msg}] {
          # Read Dockerfile
          dockerfile := file.read("Dockerfile")

          # Find all lines that start with FROM and have the latest tag
          from_lines := regex.find_n("(?m)^(FROM .*:latest|FROM --platform=[^ ]+ [^: ]+|FROM (?!scratch$)[^: ]+)( (as|AS) [^ ]+)?$", dockerfile, -1)
          from_line := from_lines[_]

          msg := sprintf("Dockerfile contains 'latest' tag in import: %s", [from_line])
        }
  # Defines the configuration for alerting on the rule
  alert:
    type: security_advisory
    security_advisory:
      severity: "medium"
```

This leverages the constraints Rego evaluation type, which will output a failure for each violation that it finds. This is handy for usability, as it will tell us exactly the lines that are not in conformance with our rules.

## Example: Security Advisories Check

This is a more complex example. Here, we'll explore a Rule Type that checks for open security advisories in a GitHub repository.

```yaml
---
version: v1
type: rule-type
name: no_open_security_advisories
context:
  provider: github
description: |
  Verifies that a repository has no open security advisories based on a given severity threshold.

  The threshold will cause the rule to fail if there are any open advisories at or above the threshold.
  It is set to `high` by default, but can be overridden by setting the `severity` parameter.
guidance: |
  Ensuring that a repository has no open security advisories helps maintain a secure codebase.

  This rule will fail if the repository has unacknowledged security advisories.
  It will also fail if the repository has no security advisories enabled.

  Security advisories that are closed or published are considered to be acknowledged.

  For more information, see the [GitHub documentation](https://docs.github.com/en/code-security/security-advisories/working-with-repository-security-advisories/about-repository-security-advisories).
def:
  in_entity: repository
  rule_schema:
    type: object
    properties:
      severity:
        type: string
        enum:
          - unknown
          - low
          - medium
          - high
          - critical
        default: high
    required:
      - severity
  ingest:
    type: rest
    rest:
      endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}/security-advisories?per_page=100&sort=updated&order=asc"
      parse: json
      fallback:
        # If we don't have advisories enabled, we'll get a 404
        - http_code: 404
          body: |
            {"fallback": true}
  eval:
    type: rego
    rego:
      type: constraints
      violation_format: json
      def: |
        package minder
        
        import future.keywords.contains
        import future.keywords.if
        import future.keywords.in
        
        severity_to_number := {
        	null: -1,
        	"unknown": -1,
        	"low": 0,
        	"medium": 1,
        	"high": 2,
        	"critical": 3,
        }
        
        default threshold := 1
        
        threshold := severity_to_number[input.profile.severity] if input.profile.severity != null
        
        above_threshold(severity, threshold) if {
        	severity_to_number[severity] >= threshold
        }
        
        had_fallback if {
        	input.ingested.fallback
        }
        
        violations contains {"msg": "Security advisories not enabled."} if {
        	had_fallback
        }
        
        violations contains {"msg": "Found open security advisories in or above threshold"} if {
        	not had_fallback
        
        	some adv in input.ingested
        
        	# Is not withdrawn
        	adv.withdrawn_at == null
        
        	adv.state != "closed"
        	adv.state != "published"
        
        	# We only care about advisories that are at or above the threshold
        	above_threshold(adv.severity, threshold)
        }
  alert:
    type: security_advisory
    security_advisory:
      severity: "medium"
```

This verifies that a repository does not have untriaged security advisories within a given severity threshold. Thus ensuring that the team is actively taking care of the advisories and publishing or closing them depending on the applicability.

## Linting

In order to enforce correctness and best practices for our rule types, we have a command-line utility called [mindev](https://github.com/stacklok/minder/tree/main/cmd/dev) that has a lint sub-command.

You can run it by doing the following from the Minder repository:

```bash
./bin/mindev ruletype lint -r path/to/rule
```

This will show you a list of suggestions to fix in your rule type definition.

The Styra team released a tool called [Regal](https://github.com/StyraInc/regal), which allows us to lint Rego policies for best practices or common issues. We embedded Regal into our own rule linting tool within mindev. So, running `mindev ruletype lint` on a rule type that leverages Rego will also show you OPA-related best practices.

Conclusion

This introductory guide provides a foundation for leveraging Rego and Minder to write policies effectively. Experiment, explore, and tailor these techniques to meet the unique requirements of your projects.

Minder is constantly evolving, so don’t be surprised if we soon add more custom functions or even more evaluation engines! The project is in full steam and more features are coming!

[You can see a list of rule types that we actively maintain here.](https://github.com/stacklok/minder-rules-and-profiles)
