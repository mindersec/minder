---
title: Writing rules using JQ
sidebar_position: 115
---

Minder's policy engine is able to use pluggable drivers for evaluating rules.
The JQ evaluator makes it easy to extract values from JSON-structured data, such
as the result of an API call.

## Writing rule types in Minder

{/* TODO: this is common between the rego and jq docs. Move somewhere common? */}

Minder
[organizes policies into rule types](../understand/key_concepts.md#rule-types),
each with specific sections defining how policies are ingested, evaluated, and
acted upon. Rule types are then called within profiles to express the security
posture of your organization. Let's delve into the essential components of a
Minder rule type:

- Ingesting data: Fetching relevant data, often from external sources like
  GitHub API.

- Evaluation: Applying policy logic to the ingested data. Minder offers a set of
  engines to evaluate data: `jq` and `rego` being general-purpose engines, while
  `vulncheck` and `homoglyphs` are more use case-specific ones.

- Remediation and alerting: Taking actions or providing notifications based on
  evaluation results. E.g. creating a pull request or generating a GitHub
  security advisory.

## JQ Evaluation

The JQ evaluator performs a series of JSON equality comparisons between ingested
JSON data and desired values provided either as a `constant` or data selected
from the `def` (`rule_schema`) of the profile. Each comparison
[must include the `ingested` value and one of the `profile` or `constant` fields](https://mindersec.github.io/ref/proto#minder-v1-RuleType-Definition-Eval-JQComparison)
for the comparison.

If a field is not present or is `null` for either the ingested data or the
desired value, it will be treated as `null` for the purposes of comparison (and
will compare equal with a `null` value).

## Example: Managing Permitted GitHub Actions

GitHub provides security controls to limit which actions can execute within a
repository. If you want to enforce a consistent set of security controls across
many repositories without paying for GitHub Enterprise, you can use Minder to
check and remediate these settings.

```yaml
---
version: v1
type: rule-type
name: allowed_selected_actions
context:
  provider: github
description: |
  Verifies the settings for selected actions and reusable workflows that are allowed
  in a repository. To use this rule, the repository profile for allowed_actions must
  be configured to selected.
guidance: |
  Ensure that only the actions and reusable workflows that are allowed
  in the repository are set.

  Having an overview over which actions and reusable workflows are
  allowed in a repository is important and allows for a better overall
  security posture.

  For more information, see [GitHub's
  documentation](https://docs.github.com/en/rest/actions/permissions#set-allowed-actions-and-reusable-workflows-for-a-repository).
def:
  # Defines the section of the pipeline the rule will appear in.
  # This will affect the template used to render multiple parts
  # of the rule.
  in_entity: repository
  # Defines the schema for writing a rule with this rule being checked
  rule_schema:
    type: object
    properties:
    properties:
      github_owned_allowed:
        type: boolean
        "description": "Whether GitHub-owned actions are allowed. For example, this includes the actions in the `actions` organization."
      verified_allowed:
        type: boolean
        "description": "Whether actions from GitHub Marketplace verified creators are allowed. Set to `true` to allow all actions by GitHub Marketplace verified creators."
      patterns_allowed:
        type: array
        description: "Specifies a list of string-matching patterns to allow specific action(s) and reusable workflow(s). Wildcards, tags, and SHAs are allowed. For example, `monalisa/octocat@*`, `monalisa/octocat@v2`, `monalisa/*`.\n\n**Note**: The `patterns_allowed` setting only applies to public repositories."
        items:
          type: string
  # Defines the configuration for ingesting data relevant for the rule
  ingest:
    type: rest
    rest:
      # This is the URL to read the actions permissions settings
      endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}/actions/permissions/selected-actions"
      parse: json
      fallback:
        # If the "github_actions_allowed" rule_type is not set to "selected", this endpoint doesn't exist and gh
        # returns a 409. Let's emit a fallback here so the evaluator fails as expected
        - http_code: 409
          body: |
            {"http_status": 404, "message": "Not Protected"}
  # Defines the configuration for evaluating data ingested against the given profile
  # Defines the configuration for evaluating data ingested against the given profile
  eval:
    type: jq
    jq:
      # Ingested points to the data retrieved in the `ingest` section
      - ingested:
          def: ".github_owned_allowed"
        # profile points to profile's rule data from .def (matching rule_schema).
        profile:
          def: '.github_owned_allowed'
      - ingested:
          def: ".verified_allowed"
        profile:
          def: '.verified_allowed'
      - ingested:
          def: ".patterns_allowed"
        profile:
          def: ".patterns_allowed"
  # Defines the configuration for remediating on the rule
  remediate:
    type: rest
    rest:
      method: PUT
      endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}/actions/permissions/selected-actions"
      # Body uses template data from the profile's rule def(initions)
      body: |
        {"github_owned_allowed":{{ .Profile.github_owned_allowed }},"verified_allowed":{{ .Profile.verified_allowed }},"patterns_allowed":[{{range $index, $pattern := .Profile.patterns_allowed}}{{if $index}},{{end}}"{{ $pattern }}"{{end}}]}
```

The
[rule evaluation details](../ref/rule_evaluation_details.md#remediation-types)
includes more information about defining remediations for Minder rules.
