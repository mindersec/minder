---
title: Register Repositories
sidebar_position: 30
---

## Register repositories

Now that you have enrolled with GitHub as a provider, you can now register repositories. We will use the `repo` command.

```bash
minder repo register --provider github 
```

You can also register a repository (or set of repositories) by name:

```bash
minder repo register --provider github --repo "owner:repo1,owner:repo2"
```

A webhook will now be created in each repository, and selected repositories will be considered registered within Mediator. Any events that now occur in any registered repository will be sent to Mediator, and processed accordingly.

## List and Get Repositories

You can list all repositories registered in Mediator:

```bash
minder repo list -n github
```

You can also get a specific repository:

```bash
minder repo get -n github -r {$repo_id}
```
