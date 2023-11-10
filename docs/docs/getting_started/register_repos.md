---
title: Register Repositories
sidebar_position: 30
---

## Prerequisites

* [The `minder` CLI application](./install_cli.md)
* [A Minder account](./login.md)
* [An enrolled GitHub token](./login.md#enrolling-the-github-provider) that is either an Owner in the organization or an Admin on the repositories

## Register repositories

Now that you have enrolled with GitHub as a provider, you can now register repositories. We will use the `repo` command.

```bash
minder repo register --provider github 
```

You can also register a repository (or set of repositories) by name:

```bash
minder repo register --provider github --repo "owner/repo1,owner/repo2"
```

A webhook will now be created in each repository that you've selected for registering with Minder.
You should see a list of the repositories that have been registered.

After registration, Minder will go through your existing profiles and apply them against these repositories.

Any events that now occur in your registered repositories will be sent to Minder and processed accordingly.

## List and Get Repositories

You can list all repositories registered in Minder:

```bash
minder repo list --provider github
```

You can also get a specific repository:

```bash
minder repo get --provider github -r {$repo_id}
```

## Deleting a registered repository

If you want to stop monitoring a repository, you can delete it from Minder by using the `repo delete` command:

```bash
minder repo delete --provider github --name "owner/repo1"
```

This will delete the repository from Minder and remove the webhook from the repository. 
