---
title: Key Concepts
sidebar_position: 05
---
# Key Concepts

## Providers

Providers are Minder’s integration points with external systems. These enable Minder to monitor and
secure supply chain components by fetching data, notifying Minder of changes, and mapping external
systems to Minder’s internal ontology.

Examples:

* GitHub and GitLab: Track repositories, pull requests, and CI/CD pipelines.

* Docker Hub: Monitor container images and their metadata.

Providers communicate with Minder through APIs, webhook events, and scheduled updates. This ensures
continuous monitoring and up-to-date information about the entities they manage.

## Entities and Checkpoints

Entities are representations of key components in the supply chain, such as repositories, pull requests, or artifacts.
They allow Minder to monitor and evaluate security practices over time.

**Checkpoints**:

Checkpoints are points in time that capture the state of an entity. These snapshots enable:

* Auditability: Tracking changes and compliance over time.

* Granular Analysis: Evaluating an entity’s state at specific moments, such as a commit hash or artifact digest.

**Entity Lifecycle**:

1. *Registration*: An entity is registered through its provider.

2. *Evaluation*: Minder evaluates the entity against defined policies.

3. *Action*: Issues identified during evaluation are either remediated or alerted.

## Policies, Rules, and Profiles

**Rules**:

Rules define individual checks for specific aspects of an entity, such as ensuring secret scanning
is enabled or that artifacts are signed.

**Profiles**:

Profiles are collections of rules tailored to a specific purpose or entity type.

## Origination

Origination describes one type of relationships between entities. For instance:

* A pull request originates from a repository.

This concept ensures that Minder maintains lifecycle consistency by:

* Automatically creating derived entities (e.g., pull requests) based on originating ones.

* Cleaning up dependent entities when the originating entity is removed.

Example Relationships:

    Repository -> Pull Request

    Repository -> Release

## Data Sources

Data sources complement providers by fetching additional contextual information from
third-party APIs. While providers manage entities, data sources provide an enhanced view
of the entity. A common example is evaluating dependencies of an entity for security risks.

Example Integration:

A data source might query the Open Source Vulnerabilities (OSV) database to check
for known vulnerabilities in dependencies listed in a repository’s manifest file.

A data source might query an external service to check for the correctness and availability
of release assets.

These differ from Providers in the sense that they do not manage entities but provide
additional context for evaluation.