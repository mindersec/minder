---
title: Quickstart with Minder (< 1 min)
sidebar_position: 20
---

# Quickstart with Minder (< 1 min)

Minder provides a "happy path" that guides you through the process of creating your first profile in Minder. In just a few seconds, you will register your repositories and enable secret scanning protection for all of them!

## Prerequisites

* A running Minder server, including a running KeyCloak installation
* A GitHub account
* [The `minder` CLI application](./install_cli.md)
* [Logged in to Minder server](./login.md)

## Quickstart

Now that you have installed your minder cli and have logged in to your Minder server, you can start using Minder!

Minder has a `quickstart` command which guides you through the process of creating your first profile.
In just a few seconds, you will register your repositories and enable secret scanning protection for all of them.
To do so, run:

```bash
minder quickstart
```

This will prompt you to enroll your provider, select the repositories you'd like, create the `secret_scanning`
rule type and create a profile which enables secret scanning for the selected repositories.

To see the status of your profile, run:

```bash
minder profile status list --name quickstart-profile --detailed
```

You should see the overall profile status and a detailed view of the rule evaluation statuses for each of your registered repositories.

Minder will continue to keep track of your repositories and will ensure to fix any drifts from the desired state by
using the `remediate` feature or alert you, if needed, using the `alert` feature.

Congratulations! 🎉 You've now successfully created your first profile!

## What's next?

You can now continue to explore Minder's features by adding or removing more repositories, create more profiles with
various rules, and much more. There's a lot more to Minder than just secret scanning.

The `secret_scanning` rule is just one of the many rule types that Minder supports.

You can see the full list of ready-to-use rules and profiles
maintained by Minder's team here - [stacklok/minder-rules-and-profiles](https://github.com/stacklok/minder-rules-and-profiles).

In case there's something you don't find there yet, Minder is designed to be extensible.
This allows for users to create their own custom rule types and profiles and ensure the specifics of their security
posture are attested to.

Now that you have everything set up, you can continue to run `minder` commands against the public instance of Minder
where you can manage your registered repositories, create profiles, rules and much more, so you can ensure your repositories are
configured consistently and securely.

For more information about `minder`, see:
* `minder` CLI commands - [Docs](https://minder-docs.stacklok.dev/ref/cli/minder).
* `minder` REST API Documentation - [Docs](https://minder-docs.stacklok.dev/ref/api).
* `minder` rules and profiles maintained by Minder's team - [GitHub](https://github.com/stacklok/minder-rules-and-profiles).
* Minder documentation - [Docs](https://minder-docs.stacklok.dev).
