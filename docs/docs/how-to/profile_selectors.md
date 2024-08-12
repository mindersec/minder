---
sidebar_position: 130
title: Apply a profile to a subset of entities
---

# Apply a profile to a subset of entities

Profiles allow you to apply a consistent set of rules to a group of entities within your project. By default, these
profiles are applied universally across all entities in a project. However, you may need to target a specific subset, such as
repositories belonging to a specific organization. Minder simplifies this process with profile selectors, enabling you
to easily customize which entities a profile applies to.

## Prerequisites

- The `minder` CLI application
- A Minder account with
  [at least `editor` permission](../user_management/user_roles.md)
- An enrolled Provider (e.g., GitHub) and registered repositories

## Add a selector to a profile

Selectors are written using [CEL (Common Expression Language)](https://github.com/google/cel-spec). To add a selector to
your profile, you need to define the entity and the condition you want to apply. Below is an example showing how to
configure a selector to filter repositories and artifacts:

```yaml
name: profile-with-selectors
selection:
  - entity: repository
    selector: repository.is_fork != true && repository.name.startsWith('stacklok/')
  - entity: artifact
    selector: artifact.provider.name == 'github-app-stacklok'
```

Let's break down the example above:
- `entity`: Defines the type of entity you want to filter (`repository`, `artifact`, or `pull_request`). In the case that the `entity` type is omitted, the selector will be applied to all entities.
- `selector`: The CEL expression that specifies the filtering criteria. In the example:
  - The first selector filters repositories to include only those that are not forks and whose name starts with stacklok. In other words, those that are part of the stacklok organization.
  - The second selector filters artifacts to include only those provided by `github-app-stacklok`.

Below you can find the full list of selectors available for each entity type.

## Repository selectors

Selectors for repositories allow you to filter and manage repositories based on specific attributes.

| Field        | Description                                                                                           | Type             |
|--------------|-------------------------------------------------------------------------------------------------------|------------------|
| `name`       | The full name of the repository, e.g. stacklok/minder                                                 | string           |
| `is_fork`    | `true` if the repository is a fork, `nil` if unknown or not applicable to this provider               | bool             |
| `is_private` | `true` if the repository is private, `nil` if unknown or not applicable to this provider              | bool             |
| `provider`   | The provider of the repository, for more details see [Provider Selectors](#entity-provider-selectors) | ProviderSelector |

## Artifact selectors

| Field      | Description                                                                                         | Type             |
|------------|-----------------------------------------------------------------------------------------------------|------------------|
| `name`     | The full name of the artifact, e.g. stacklok/minder-server                                          | string           |
| `type`     | The type of the artifact, e.g. "container"                                                          | string           |
| `provider` | The provider of the artifact, for more details see [Provider Selectors](#entity-provider-selectors) | ProviderSelector |

## Pull request selectors

| Field  | Description                                                 | Type   |
|--------|-------------------------------------------------------------|--------|
| `name` | The full name of the pull request, e.g. stacklok/minder/123 | string |

## Entity provider selectors

Each entity can be filtered based on its provider.

| Field   | Description                                        | Type   |
|---------|----------------------------------------------------|--------|
| `name`  | The name of the provider, e.g. github-app-stacklok | string |
| `class` | The class of the provider, e.g. github-app         | string |