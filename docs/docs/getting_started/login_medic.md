---
title: Login with Minder
sidebar_position: 20
---

## Prerequisites

* The `minder` CLI application

## Login as a user

1. Log in with username and password.  By default, `minder` will run against a public stacklok cloud instance, but this can be changed in `config.yaml` in your local directory or using the `--gprc-host` and `--grpc-port` flags.

```bash
minder auth login
```

A new browser window will open, where you can register a new user and log in.

Once logged in

2. Enroll a user with the given provider

```bash
minder provider enroll --provider github --group-id <group-id>
```

> __Note__: Provide the `--group-id` flag, if your user belongs to multiple groups. For this example, we will use the default group `1`, so we do not need to provide the flag.
> 
A browser session will open, and you will be prompted to login to your GitHub. Once you have granted mediator access, you will be redirected back, and the user will be enrolled. The `minder` CLI application will report the session is complete.
