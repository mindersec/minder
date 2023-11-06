---
title: Enabling Dependabot
sidebar_position: 30
---

# Ensuring Dependabot is Enabled for Repositories

Minder allows you to consistently enforce security policies on your GitHub repositories.  In this tutorial, you'll use Minder to ensure that Dependabot is enabled on your enrolled GitHub repositories.

## Prerequisites

* [The `minder` CLI application](./install_cli.md)
* [A Minder account](./login.md)
* A running Minder server
* A GitHub account

## Login as a user



1. Log in with username and password.  By default, `minder` will run against a public Stacklok cloud instance, but this can be changed in `config.yaml` in your local directory or using the `--gprc-host` and `--grpc-port` flags.

```bash
minder auth login
```

A new browser window will open, where you can register a new user and log in.

Once logged in:

2. Enroll a user with the given provider

```bash
minder provider enroll --provider github
```

A browser session will open, and you will be prompted to login to your GitHub account. Once you have granted Minder access, you will be redirected back, and the user will be enrolled. The `minder` CLI application will report the session is complete.
