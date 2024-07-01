---
title: Quickstart with Minder (< 1 min)
sidebar_position: 30
---

# Quickstart with Minder (< 1 min)

Minder provides a straightforward "quickstart" functionality that will create your first profile in Minder which ensures that GitHub secret scanning is enabled, and lets you select the GitHub repositories that you want this profile to apply to.

The quickstart lets you get started with Minder, ensuring that secret scanning is enabled for your repositories in seconds.

## Prerequisites

Before you can run the `quickstart` command, you must [log in to Minder using the CLI](login).

## Quickstart

Now that you have installed the Minder CLI and have logged in to your Minder server, you can start using Minder!

Minder's `quickstart` command will simplify getting started managing GitHub repositories. It will perform three steps to help you get started:

* [enroll the GitHub provider](enroll_provider), so that Minder can access your repositories
* [register repositories](register_repos), which selects which repositories you want to manage
* [add a rule type and create a profile](first_profile), which will detect repositories that don't have secret scanning enabled.

To get started, run:

```bash
minder quickstart
```

#### Enrolling the GitHub provider

This first step configures GitHub and produces an authentication token that allows Minder to inspect and manage your repository configuration. You will be prompted to grant Minder access.

#### Registering repositories

This step allows you to select the repositories that you want Minder to manage. Every repository that you select will be scanned according to the profile that quickstart will set up (next). This profile will ensure that [secret scanning](https://docs.github.com/en/code-security/secret-scanning/about-secret-scanning) is enabled for these repositories.

#### Create the `secret_scanning` rule type

This step will upload the `secret_scanning` rule type to the server.

A _rule type_ is a definition of an individual security setting and how to evaluate it; for example, the `secret_scanning` rule type contains the logic to query GitHub and evaluate whether secret scanning is enabled for an individual repository.

Minder allows you to build custom rule types, or use [one of our pre-defined rule types](https://github.com/stacklok/minder-rules-and-profiles/pulls). But in either case, these rules must be uploaded to the Minder server before you can use them.

#### Create the `quickstart` profile

This step will create a profile named `quickstart-profile` that contains the `secret_scanning` rule type.

A _security profile_ is a definition of the rule types that you want to apply to your repositories. The `quickstart` command will create a profile with a single rule type, the `secret_scanning` rule type that it uploads. Once this has been created, Minder will scan all the repositories that you selected in step two to ensure that secret scanning is enabled for each of them.

Congratulations! 🎉 You've now successfully created your first profile!

## See the status of your profile

To see the status of your profile, run:

```bash
minder profile status list --name quickstart-profile --detailed
```

This command shows you the overall status of your profile, and how each rule evaluates for each of your registered repositories.

You should see an entry for each repository that you registered. If the repository has secret scanning enabled, you should see a status of "Success"; if the repository does _not_ have secret scanning enabled, you should see a status of "Failure".

## What's next?

There's a lot more to Minder than just secret scanning!

Now that you have everything set up, you can continue to run `minder` commands against the public instance of Minder
where you can manage your registered repositories, create profiles, rules and much more, so you can ensure your repositories are
configured consistently and securely.

* [Register more repositories](register_repos) to take advantage of Minder for more of your organization
* [Add additional rules and profiles](first_profile) to define your full security profile for your organization; you can see all of Minder's ready-to-use rules and example profiles [on GitHub](https://github.com/stacklok/minder-rules-and-profiles).

In case there's something you don't find there yet, Minder is designed to be extensible. This allows for users to create their own custom rule types and profiles and ensure the specifics of their security posture are attested to.

## More information

For more information about `minder`, see:
* `minder` CLI commands - [Docs](https://minder-docs.stacklok.dev/ref/cli/minder).
* `minder` REST API Documentation - [Docs](https://minder-docs.stacklok.dev/ref/api).
* `minder` rules and profiles maintained by Minder's team - [GitHub](https://github.com/stacklok/minder-rules-and-profiles).
* Minder documentation - [Docs](https://minder-docs.stacklok.dev).
