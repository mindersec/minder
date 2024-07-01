---
title: Registering repositories
sidebar_position: 50
---

Once you have enrolled the GitHub Provider, you can register your GitHub repositories with your Minder organization. This will define the repositories that your security profile will apply to.

## Prerequisites

Before you can register a repository, you must [enroll the GitHub Provider](enroll_provider).

## Register repositories

Once you have enrolled the GitHub Provider, you can register repositories that you granted Minder access to within GitHub.

To get a list of repositories, and select them using a menu in Minder's text user interface, run:

```bash
minder repo register
```

You can also register an individual repository by name, or a set of repositories, comma-separated. For example:

```bash
minder repo register --name "owner/repo1,owner/repo2"
```

After registering repositories, Minder will begin applying your existing profiles to those repositories and will identify repositories that are out of compliance with your security profiles.

In addition, Minder will set up a webhook in each repository that was registered. This allows Minder to identify when configuration changes are made to your repositories and re-scan them for compliance with your profiles.

## More information

For more information about repository registration, see the [additional documentaiton in "How Minder works"](../understand/repository_registration).
