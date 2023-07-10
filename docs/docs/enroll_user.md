---
id: enroll_user
title: Getting Started (Enroll User & Register Repositories)
sidebar_position: 4
slug: /enroll_user
displayed_sidebar: mediator
---

# Getting Started (Enroll User)

Now that you have [configured your OAuth provider](./config_oauth), you can enroll a user into Mediator.

## Prerequisites

* The `medic` CLI application
* A [running mediator instance](./get_started)
* [OAuth Configured](./config_oauth)

## Login as a user

1. Log in with username and password.  By default, `medic` will run against localhost, but this can be changed in `config.yaml` in your local directory.

```bash
medic auth login --username root --password password
```

> __Note__: The default username and password are `root` and `P4ssw@rd` respectively. You should change these immediately.

2. Enroll a user with the given provider

```bash
medic  enroll provider --provider github
```

> __Note__: Provide the `--group-id` flag, if your user belongs to multiple groups. For this example, we will use the default group `1`, so we do not need to provide the flag.
> 
A browser session will open, and you will be prompted to login to your GitHub. Once you have granted mediator access, you will be redirected back, and the user will be enrolled. The `medic` CLI application will report the session is complete.

## Register repositories

Now that you have enrolled with GitHub as a provider, you can now register repositories. We will use the `repo` command.

```bash
medic repo register --provider github 
```

You can also register a repository (or set of repositories) by name:

```bash
medic repo register --provider github --repo "owner:repo1,owner:repo2"
```

A webhook will now be created in each repository, and selected repositories will be considered registered within Mediator. Any events that now occur in any registered repository will be sent to Mediator, and processed accordingly.

## List and Get Repositories

You can list all repositories registered in Mediator:

```bash
medic repo list -n github
```

You can also get a specific repository:

```bash
medic repo get -n github -r {$repo_id}
```
