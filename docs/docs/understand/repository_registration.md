---
title: Repository registration
sidebar_position: 50
---

_Registering a repository_ tells Minder to apply the [profiles](profiles.md) that you've defined to that repository. Minder will continuously monitor that repository based on the profiles that you've defined, and optionally [alert you](alerts.md) or [automatically remediate the problem](remediations.md) when the repository is out of compliance.

## Registering repositories

Once you have [enrolled the GitHub Provider](providers.md), you can register repositories that you granted Minder access to within GitHub.

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

## Automatically registering new repositories

The GitHub Provider can be configured to automatically register new repositories that are created in your organization. This is done by setting an attribute on the provider.

First, identify the _name_ of your GitHub Provider. You can list your enrolled providers by running:

```bash
minder provider list
```

To enable automatic registration for your future repositories, set the `auto_registration.entities.repository.enabled` attribute to `true` for your provider. For example, if your provider was named `github-app-myorg`, run:

```bash
minder provider update --set-attribute=auto_registration.entities.repository.enabled=true --name=github-app-myorg
```

To enable automatic registration for existing repositories and enroll all the currently existing repositories, you can use the `minder repo register` command:
```bash
minder repo register --all
```
You can pass the `--provider` flag to restrict the registration to a specific provider. By default, the `repo register` command will register repositories for all providers.

To disable automatic registration, set the `auto_registration.entities.repository.enabled` attribute to `false`:

```bash
minder provider update --set-attribute=auto_registration.entities.repository.enabled=false --name=github-app-myorg
```

:::note
Disabling automatic registration will not remove the repositories that have already been registered.
:::

## List and get Repositories

You can list all repositories registered in Minder:

```bash
minder repo list
```

You can also get detailed information about a specific repository. For example, to view the information for `owner/repo1`, run:

```bash
minder repo get --name owner/repo1
```

## Removing a registered repository

If you want to stop monitoring a repository, you can remove it from Minder by using the `repo delete` command:

```bash
minder repo delete --name "owner/repo1"
```

This will remove the repository configuration from Minder and remove the webhook from the GitHub repository. 
