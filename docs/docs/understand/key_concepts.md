---
title: Key Concepts
sidebar_position: 05
---

Minder implements a platform for enforcing supply chain security policy in a
_continuous_ and _automated_ manner. In addition to policy checks, Minder also
supports defining _remediation_ actions that can be automatically executed to
assist teams in following the defined policies. This section introduces the key
concepts in Minder for defining _what_ policies should be applied to _which_
resources, and how Minder uses these concepts to enforce security policies.

## Managing supply chains with Minder

### Projects

Projects are the unit of tenancy (separation and control of resources by
different users) in Minder. Projects are used to group supply chain components
which are managed by a common team, and to apply policies to those components.
One user may be a member of multiple projects, and one project may be managed by
multiple users.

Users can be assigned a [role](../user_management/user_roles.md) in a project,
which determines their permissions to view and manage the project's resources,
such as [entities](#entities), [providers](#providers), and
[profiles](#profiles).

### Entities

Entities represent components in the supply chain, such as repositories, pull
requests, or artifacts. Minder uses entities to track which supply chain
components are associated with which policies and rules, which guides
[rule evaluation](#phases-of-evaluation) when it occurs. In addition to an
intrinsic identifier (such as GitHub repo name), entities have a set of
system-provided _properties_ which are extracted from the underlying system and
can be used when evaluating policies.

Entities are created and managed by providers.

### Providers

Providers are Minder’s integration points with external systems, such as GitHub
or the Docker registry. Providers track the credentials and permissions needed
to interact with these services, and enable both manual and automatic creation
of entities, depending on the entity type and provider configuration. In
general, references to an entity need to be _qualified_ by the context of the
provider that created the entity, though the Minder API will attempt to deduce
the appropriate provider where possible.

Examples:

- GitHub and GitLab: Track repositories, pull requests, and CI/CD pipelines.

- Docker Hub: Monitor container images and their metadata.

Providers communicate with Minder through APIs, webhook events, and scheduled
updates. This ensures continuous monitoring and up-to-date information about the
entities they manage.

#### Origination

Some entities are automatically created by a provider due to existing
relationships in the external system that the provider interacts with.
Origination is the term used to describe entities which have been automatically
created due to their relationship with an existing entity. For instance, a pull
request originates from a repository.

This concept ensures that Minder maintains lifecycle consistency by:

- Automatically creating derived entities (e.g., pull requests) based on
  originating ones.

- Deleting dependent entities when the originating entity is removed.

Example Relationships:

- Repository -> Pull Request

- Repository -> Release

### Profiles

Profiles represent a collection of individual controls or policies which
collectively enforce a security posture or other requirements on a set of
entities. Profiles contain a collection of [rule types](#rule-types) with
parameters (such as permitted CVE severity or allowed license types) to control
the execution of the rule. Best practice is to define profiles that apply a set
of related behaviors, and define your desired security posture via the
application of multiple profiles.

Profiles are specific to each project, but can apply to entities across multiple
providers with the project. While profiles apply to all entities in a project by
default, profiles may contain a [selector](../how-to/profile_selectors.md) which
limits to the profile to only entities matched by the selector expression.

#### Rule types

Rule types define individual checks for specific aspects of an entity, such as
ensuring secret scanning is enabled or that artifacts are signed. While a rule
type defines a specific check on an entity, rule types may also contain
parameters which can be set by the profile which applies the rule type to the
selected set of entities. In this way, a single rule type (for example,
requiring GitHub Actions configuration) can be parameterized for different
programming languages, licenses, or repository visibility.

Like profiles, rule types are specific to a project, but the same definition can
be shared and loaded into multiple projects. A collection of useful rule
definitions is available in
https://github.com/mindersec/minder-rules-and-profiles.

## Executing policy with Minder

### Phases of evaluation

Minder attempts to ensure that entities are continuously against the defined
polices. It does this by performing rule evaluation at various times, including
when entities are first registered, when notified of a change to an entity, and
(soon) periodically to catch changes which are not notified. When executing the
rules from a policy, Minder proceeds to evaluate all the rules in the relevant
policies in parallel using the following phases:

1. **Ingestion**: Fetch the latest state of the entity from the provider.
2. **Evaluation**: Evaluate the rules against the entity.
3. **Remediation**: If a rule fails, attempt to remediate the entity.
4. **Alert**: If a rule fails and remediation is not possible, create an alert.

The details of rule evaluation are covered
[in a separate document](./rule_evaluation.md); this section provides a high
level overview to complement the policy management constructs described in
[managing supply chains](#managing-supply-chains-with-minder).

### Ingesters

Ingesters fetch data about the entity using provider-specific code. Depending on
the type of entity, this data might include results from API calls, file
contents, or other data such as attestations. Generally, data from the ingestion
phase will be made available as either structured data (such as a JSON document)
or through Rego functions in the Rego evaluation engine.

### Evaluation engines

The rule evaluation engine is at the heart of defining policies on supply chain
entities. It allows rule type authors to compare the data fetched during the
ingestion stage with expected values, and determine whether the entity meets the
policy requirements.

Minder currently supports two rule evaluation engines: `rego` and `jq`. The `jq`
engine is useful for evaluating simple expressions against constant or
parameterized values, while the `rego` engine is more powerful, and allows
writing expressions with conditionals, loops and dynamic data fetching.

At the end of the rule evaluation, each rule type yields a result, which can be
one of four values:

- `pass`: The entity meets the policy requirements.
- `fail`: The entity does not meet the policy requirements.
- `skip`: The rule was skipped because it did not apply to the entity.
- `error`: An error occurred during the evaluation.

#### Data sources

Data sources complement providers by fetching additional contextual information
from third-party APIs. Data sources are currently only available in the Rego
evaluation engine. While providers manage entities and can supply data during
the ingestion phase, data sources provide a structured interface to data sources
which can be queried dynamically by the rule engine..

Like rule types and profiles, data sources are defined in the context of a
project. Data types generally fetch data from an external network service, and
can be used to enrich the data extracted by the rule engine:

- A data source might query the Open Source Vulnerabilities (OSV) database to
  check for known vulnerabilities in dependencies listed in a repository’s
  manifest file.

- A data source might query an external service to check for the correctness and
  availability of release assets.

- A data source might be used to fetch additional information from the same API
  as the provider to compare data from two different sources, such as the list
  of branches and the branch protection rules.

#### Remediations and alerts

For rule types which produce a `fail` result for an entity, remediations and
alerts define corrective actions which Minder can take to restore the entity to
a policy-compliant state. Remediations and alerts may each be defined as part of
the rule type; some rule may define only a remediation, only an alert, both, or
neither.

Generally, remediations define actions which Minder can take which will directly
address the identified issue -- for example, updating an entity via a REST API
or proposing a change through a pull request. Alerts provide a mechanism for
providing feedback to humans about the non-compliant state of an entity; while
alerts may provide detailed advice on how to correct a problem, they do not
correct the problem on their own.

While rule types define the remediation and alert mechanisms, policies can
enable or disable the execution of either remediations or alerts. This allows
policy authors to begin implementing rules by measuring policy compliance before
adding remediation or alert actions which may disrupt the workflow of
developers.

#### Historical evaluation records

In addition to remediation and alert actions, Minder also maintains a historical
evaluation record for each rule. This record includes information about when the
rule was evaluated, the evaluation result, and any messages and actions taken as
a result of the rule evaluation.
