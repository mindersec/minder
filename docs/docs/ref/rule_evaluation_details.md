# Rule Evaluation in Depth

When evaluating a rule,
[Minder executes the rule in multiple phases](https://mindersec.github.io/understand/key_concepts#phases-of-evaluation).
This page contains detailed documentation about the execution of each phase.
Minder performs rule evaluation across all relevant policies when it detects
that an entity has changed.

## Ingest

The ingest phase in Minder is where the system gathers and processes information
about entities before policy evaluation. Here's a breakdown:

### Input Sources:

The primary input is the `Entity` object representing the entity that the policy
rules apply to. The `Entity` objects have specific properties based on the type
of entity:

- [Repository](../ref/proto.mdx#minder-v1-Repository)
- [Pull Request](../../internal/providers/github/properties/pull_request.go)
- ??? what other types? Artifact?

### Ingest Types and Their Outputs

Each rule can specify the type of data which should be ingested. Minder works to
cache ingested data across all rule evaluations, so multiple rules which ingest
the same data should only need to fetch it once. The ingestion types produce the
following data for rule evaluatino:

1. **REST Ingest** (`rest`)

   _Entity Types_: all types

   _Data Content_: Calls a REST endpoint specified in the ingest type, and
   returns a parsed JSON object from the response

1. **Git Ingest** (`git`)

   _Entity_Types_: PRs and repos

   _Data Content_: This is only useful with the Rego evaluation engine. It
   provides a filesystem view of the current repository contents; this is
   required for using the Rego `fs` methods.

1. **Dependency Ingest** (`deps`)

   _Entity_Types_: PRs and repos

   _Data Content_: Ingests the set of libraries used by recognized package
   managers, using the [osv-scalibr](https://github.com/google/osv-scalibr)
   library. When used to ingest a pull request, the `deps` ingest can provide
   `new`, `new_and_updated`, or `all` dependencies from the repository after the
   proposed change.

1. **Artifact Ingest** (`artifact`)

   _Entity_Types_: artifact

   _Data Content_: Fetches details about about container build attestations,
   including signature data, branch and repository information, and GitHub
   runner environment.

1. **Diff Ingest** (`diff`)

   _Entity_Types_: PR only

   _Data Content_: Provides a set of diffs describing the changes proposed in
   the PR. This works with the `homoglyphs` and `vulncheck` evaluators. The diff
   can evaluate `full` (all files) and `dep` (dependency changes) for these
   evaluators.

   <!--, and `new-dep` (which uses a more sophisticated extraction method using the [osv-scalibr library](https://github.com/google/osv-scalibr)). -->

<!--
1. **Ingest** (`builtin`)

   _Entity_Types_:

   _Data Content_:  calls builtin function (none exist since https://github.com/mindersec/minder/pull/760)
-->

## Evaluate

The evaluation phase in Minder evaluates the ingested data according to the
[specified rule type and parameters](https://mindersec.github.io/ref/proto#minder-v1-RuleType-Definition-Eval).
Here's how it works:

### Evaluation Process

Minder takes the following inputs for rule evaluation:

- Ingested data from the [ingest phase](#ingest)
- Rule profile parameters from the `params` and `def` clauses.
- Entity properties as string key-value pairs (e.g.,
  `github/default_branch=main`)

The system then evaluates the rule using one of these engines:

1. **Rego Evaluation** (`rego`)

   See the
   [documentation on writing rules for Rego](../how-to/writing-rules-in-rego.md)
   for more details on the Rego evaluation engine. This engine has the most
   flexibility and supports sophisticated logic, but also has a higher learning
   curve.

   - Uses [Open Policy Agent](https://www.openpolicyagent.org/) (OPA) Rego
     language
   - Supports two evaluation modes:
     - `deny-by-default`: Rule fails unless `allow := true` is set
     - `constraints`: Rule fails if the `violations` array contains any entries.
       `violations` elements must be an object which contains a `msg` key, e.g.
       `violations := [{"msg": "..."}, {"msg": "..."}]`
   - Produces output via the `output` property in the Rego code

1. **JQ Evaluation** (`jq`)

   See the
   [documentation on writing rules in JQ](../how-to/writing-rules-in-jq.md) for
   more details on the JQ evaluation engine. This engine allows you to easily
   extract structured data and compare it with known or expected values.

   - Processes JSON data using [jq queries](https://stedolan.github.io/jq/)
   - Returns evaluation results based on the query output

1. **Vulncheck Evaluation** (`vulncheck`)

   See the
   [documentation on checking vulnerabilities](../integrations/community_integrations.md)
   for more details on the vulncheck engine. The vulncheck engine is a
   custom-coded engine which evaluates software dependencies against the
   [Open Source Vulnerabilities (OSV) database](https://osv.dev/). With
   improvements to the `deps` ingestion type, this evaluator can largely be
   replaced with rego evaluation and
   [data sources](../understand/data_sources.md).

   - Requires the use of the `diff` ingestion type with specified `ecosystems`,
     and only operates in a `pull_request` context
   - Uses a custom parser for dependency files, and can suggest line-level
     comments on vulnerable libraries
   - Immediately applies comments highlighting new vulnerable libraries when
     evaluated against a pull request.

1. **Homoglyph Evaluation** (`homoglyph`)

   This rule evaluation engine attempts to detect malicious Unicode sequences as
   described in the [Trojan Source attack](https://trojansource.codes/). The
   homoglyphs evaluator can detect two different types of attack:

   - `invisible_characters`: using byte order characters to attempt to
     confusingly display characters and comments
   - `mixed_scripts`: mixing identical-appearance characters from different
     alphabets (for example, to use two variables with seemingly-identical
     names)

   The `homoglyph` evaluator only operates in a `pull_request` context using a
   `full` diff.

The evaluation engine determines if the rule passes, fails or should be skipped
(for example, because the resource is not the correct type). If rule passes or
is skipped, the entity is compliant with the profile, and no further evaluation
is done. If the rule evaluation fails and remediation or alerting is enabled,
output data from the rule evalution may be passed to the following steps.

## Remediate

A rule can optionally define a
[remediation action](https://mindersec.github.io/ref/proto#minder-v1-RuleType-Definition-Remediate)
to take when rule evaluation fails. The goal of remediation is to change the
state of the entity so that it it now passes the rule evaluation. The
remediation phase can optionally use inputs from the evaluation phase to
determine what actions to take.

### Remediation Inputs

Remediation actions have access to:

- The `Entity` object representing the resource being evaluated
- The `Profile` object containing rule parameters and definitions
- For remediations which create a pull request, output from the rule evaluation
  is available in `EvalResultOutput`.

### Remediation Types

Minder supports three remediation actions:

1. **Pull Request** (`pull_request`)

   The Pull Request remediation type creates or updates GitHub pull requests to
   implement fixes automatically. The pull requests are authored by the Minder
   bot, and will need to be reviewed and merged by the project maintainers.

   Pull requests support the following content modification types:

   - `minder.content`:
     [define a list of file `path`s and `content`s](https://mindersec.github.io/ref/proto#minder-v1-RuleType-Definition-Remediate-PullRequestRemediation-Content)
     which should be updated. Currently only supports the `replace` action on
     files. Both `path` and `content` may use Go templates to parameterize their
     outputs
   - `minder.actions.replace_tags_with_sha`: uses
     [the Stacklok/frizbee library](https://github.com/Stacklok/frizbee) to
     resolve Git and OCI tag references to SHA digests where detected in the
     repository. The only parameter to this action is a list of resources which
     should not be resolved to digests (implicitly trusting the release process
     for the resource)
   - `minder.yq.evaluate`: applies a
     [yq `expression`](https://mikefarah.gitbook.io/yq/) to files selected by a
     list of `patterns`. Each element in the `patterns` list is represented as:
     `{"type": "glob", "pattern": "file/path/*"}`. `glob` is currently the only
     supported pattern type. This action does not support templating either
     `expression` or `pattern`

   If the content modification produces a diff in the repository, Minder will
   open and manage a pull request against the branch used in the `git` ingest,
   or the default branch if a different ingestion was used. The pull request
   includes the following fields which support Go templates:

   - `title`: PR title
   - `body`: PR description

   The following data is available to fill in template contents in title, body,
   and for the `minder.content` action:

   - `Entity` contains the same entity information available during rule
     evaluation
   - `Profile` contains the profile data supplied in the `def` field
   - `Params` contains the profile data supplied in the `params` field
   - `EvalResultOutput` contains the output data from the rule evaluation step

2. **REST Call** (`rest`)

   The
   [REST remediation](https://mindersec.github.io/ref/proto#minder-v1-RestType)
   calls the specified `endpoint` using the defined HTTP `method`, passing a
   `body` and optional `headers`.

   Both the endpoint and the body support Go template parameters, with the
   following data:

   - `Entity` contains the same entity information available during rule
     evaluation
   - `Profile` contains the profile data supplied in the `def` field
   - `Params` contains the profile data supplied in the `params` field

3. **GitHub Branch Protection** (`gh_branch_protect`)

   The
   [branch protection remediation](https://mindersec.github.io/ref/proto#minder-v1-RuleType-Definition-Remediate-GhBranchProtectionType)
   takes a Go-templated JSON object in `patch` to merge with the existing
   protection branch protection settings. Branch protection has been implemented
   as a special remediation due to the peculiarities of the GitHub branch
   protection API.
   [Repository rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets)
   are a newer feature which avoids many of these peculiarities.

   The `patch` string supports Go template parameters and needs to output a JSON
   object which will be merged with the existing branch protection settings. The
   following data is available within the Go template context:

   - `Entity` contains the same entity information available during rule
     evaluation
   - `Profile` contains the profile data supplied in the `def` field
   - `Params` contains the profile data supplied in the `params` field

API-driven remediations (`rest` and `gh_branch_protect`) will generally take
effect immediately on the targeted entity; `pull_request` remediations will need
to be merged before they take effect. Minder will ensure that at most one pull
request is open at a time for a particular rule applied to a specific entity.

## Alert Types

When remediation has completed, Minder will execute any
[alerts defined for the rule type](https://mindersec.github.io/ref/proto#minder-v1-RuleType-Definition-Alert).
Minder currently defines two alert types, which operate similarly to the
[remediation actions](#remediation-types) except that alerts do not directly
attempt to fix the detected problem, but rather notify human maintainers to
allow them to evaluate the solution.

### Alert Inputs

Because alerts are primarily indended to drive later human behavior, they have
limited processing functionality. Alerts have access to:

- Whether a remediation rule is defined for `security_advisory` alerts
- Output from the rule evaluation as `EvalResultOutput` for
  `pull_request_comment`s

### Alert Types

1. **Pull Request Comments** (`pull_request_comment`)

   Instructs Minder to comment on new and updated pull requests. Minder will
   ensure that each PR in a repository has at most one comment per rule type
   evaluated, and will delete the PR comment if the pull request rule evaluation
   succeeds in the future (for example, because the flagged issue was
   addressed).

   Pull request comments have a single `review_message` parameter, which
   contains a Go templated markdown string with the following contents from the
   rule evaluation:

   - `EvalErrorDetails` contains any detailed error messages from the rule
     evaluation execution
   - `EvalResultOutput` contains the output data from the rule evaluation step

   The `pull_request_comment` alert type is only valid on pull request entities.

2. **GitHub Security Alerts** (`security_alert`)

   This rule instructs Minder to create a
   [private security vulnerability report](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability)
   through the GitHub API, assuming that private vulnerability reporting is
   enabled for the repository. These reports require administrator permission to
   view. When the security alert is no longer active because the rule evaluation
   passes or is skipped, Minder will automatically close the security
   vulnerability.

   The contents of the security advisory are currently hard-coded, and include
   the following details:

   - Rule evaluation error message
   - Repository name
   - Profile and rule name which was
   - Rule severity from the rule type definition
   - Any [`guidance`] content from the rule type definition
