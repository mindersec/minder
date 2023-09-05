---
id: login_medic
title: Login with Medic
sidebar_position: 1
slug: /run-the-login_medic
displayed_sidebar: mediator
---

## Prerequisites

* The `medic` CLI application

## Login as a user

1. Log in with username and password.  By default, `medic` will run against a public stacklok cloud instance, but this can be changed in `config.yaml` in your local directory or using the `--gprc-host` and `--grpc-port` flags.

```bash
medic auth login --username  <username> --password <password>
```

Once logged

2. Enroll a user with the given provider

```bash
medic provider enroll --provider github --group-id <group-id>
```

> __Note__: Provide the `--group-id` flag, if your user belongs to multiple groups. For this example, we will use the default group `1`, so we do not need to provide the flag.
> 
A browser session will open, and you will be prompted to login to your GitHub. Once you have granted mediator access, you will be redirected back, and the user will be enrolled. The `medic` CLI application will report the session is complete.
