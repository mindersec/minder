---
title: Creating a New Project
sidebar_position: 90
---

When you log in to Minder without a project, Minder will automatically create a
new project to manage your entities (repositories, artifacts, etc). It is also
possible to create additional projects after you have created your Minder
profile, for example, to create different projects for different organizations
or teams you are a part of. Note that
[Minder Projects](../understand/projects.md) can collect resources from several
upstream resource providers such as different GitHub organizations, so you can
register several [entity providers](../understand/providers.md) within a
project.

## Prerequisites

- A Minder account
- A GitHub organization you are an administrator for which does not have the
  Minder app installed on.

## Creating a New Project

To create a new project, enable the Minder GitHub application
([Minder by Stacklok](https://github.com/apps/minder-by-stacklok) for the
cloud-hosted application) on a new GitHub organization. If the GitHub App is
installed on a GitHub organization which is not already registered in Minder,
Minder will create a new project to manage those resources. Using
[`minder provider enroll`](../ref/cli/minder_provider_enroll.md) within a
project to add a new GitHub provider will _not_ create a new project and will
add the selected organization to an existing project.
