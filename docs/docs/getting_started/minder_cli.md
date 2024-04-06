---
title: Getting Started with Minder's CLI
sidebar_position: 10
---

- [Installing the Minder CLI](#installing-the-minder-cli)
- [Minder CLI configuration](#minder-cli-configuration)
- [Quickstart with Minder](#quickstart-with-minder)

<br/>

# Installing the Minder CLI

Minder consists of two components: a server-side application, and the `minder`
CLI application for interacting with the server.  Minder is built for `amd64`
and `arm64` architectures on Windows, MacOS, and Linux.

You can install `minder` using one of the following methods:

## MacOS (Homebrew)

The easiest way to install `minder` is through [Homebrew](https://brew.sh/):

```bash
brew install stacklok/tap/minder
```

Alternatively, you can [download a `.tar.gz` release](https://github.com/stacklok/minder/releases) and unpack it with the following:

```bash
tar -xzf minder_${RELEASE}_darwin_${ARCH}.tar.gz minder
xattr -d com.apple.quarantine minder
```

## Windows (Winget)

For Windows, the built-in `winget` tool is the simplest way to install `minder`:

```bash
winget install stacklok.minder
```

Alternatively, you can [download a zipfile containing the `minder` CLI](https://github.com/stacklok/minder/releases) and install the binary yourself.

## Linux

We provide pre-built static binaries for Linux at: https://github.com/stacklok/minder/releases.

## Building from source

You can also build the `minder` CLI from source using `go install github.com/stacklok/minder/cmd/cli@latest`, or by [following the build instructions in the repository](https://github.com/stacklok/minder#build-from-source).

---

# Minder CLI configuration

The Minder CLI application is configured using a YAML file. The default location for the configuration file
is `$PWD/config.yaml`. You can specify a different location using the `--config` flag. If there's no configuration 
file at the specified location, the CLI application will use its default values.

## Prerequisites

* The `minder` CLI application
* A Stacklok account

## Configuration file example

Below is an example configuration file. The `grpc_server` section configures the gRPC server that the CLI
application will connect to. The `identity` section configures the issuer URL and client ID for the
Stacklok Identity service.

```yaml
---
# Minder CLI configuration
# gRPC server configuration
grpc_server:
  host: "127.0.0.1"
  port: 8090

identity:
  cli:
    issuer_url: http://localhost:8081
    client_id: minder-cli
---
```

## Handle multiple contexts using a configuration file 

The Minder CLI can be configured to use multiple contexts. A context is a set of configuration values that
are used to define a context, i.e. connect to a specific Minder server. For example, you may have a context for your local
development environment, a context for your staging environment, and a context for your production
environment. You can also specify things like the default `provider`, `project` or preferred format `output`
for each of those.

To create a new context, create a new configuration file and set the `MINDER_CONFIG` environment variable
to point to the config file.  For a single command, you can alsothe path to the file through the `--config`
flag . For example, you can create your staging configuration in `config-staging.yaml` and use it as either:

```bash
export MINDER_CONFIG=./config-staging.yaml
minder auth login
# OR:
minder auth login --config ./config-staging.yaml
```

---

# Quickstart with Minder 

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
