---
title: Register Repositories
sidebar_position: 50
---

## Prerequisites

* [The `minder` CLI application](./install_cli.md)
* [A Minder account](./login.md)
* [An enrolled GitHub token](./login.md#enrolling-the-github-provider) that is either an Owner in the organization or an Admin on the repositories

## Register repositories

Now that you have enrolled with GitHub as a provider, you can now register repositories. We will use the `repo` command.

```bash
minder repo register
```

You can also register a repository (or set of repositories) by name:

```bash
minder repo register --name "owner/repo1,owner/repo2"
```

A webhook will now be created in each repository that you've selected for registering with Minder.
You should see a list of the repositories that have been registered.

After registration, Minder will go through your existing profiles and apply them against these repositories.

Any events that now occur in your registered repositories will be sent to Minder and processed accordingly.

## Automatically Registering Repositories

The GitHub provider can also be configured to automatically register repositories. This is done by setting the `auto_registration.entities.repository.enablede` field to `true` in the provider configuration:

```bash
minder provider update --set-attribute=auto_registration.entities.repository.enabled=true --name=github-app-myorg
```

You can list your enrolled providers with:
```bash
minder provider list
```

Note that enabling the auto-registration will merely register repositories as
they are created, not register already existing repositories.

To disable automatic registration, set the `auto_registration.entities.repository.enabled` field to `false`:
```bash
minder provider update --set-attribute=auto_registration.entities.repository.enabled=false --name=github-app-myorg
```

Note that disabling the automatic registration will not remove the repositories that have already been registered.

## List and Get Repositories

You can list all repositories registered in Minder:

```bash
minder repo list
```

You can also get a specific repository:

```bash
minder repo get --id {ID}
```

## Deleting a registered repository

If you want to stop monitoring a repository, you can delete it from Minder by using the `repo delete` command:

```bash
minder repo delete --name "owner/repo1"
```

This will delete the repository from Minder and remove the webhook from the repository. 
