![minder logo](./docs/docs/images/Minder_darkMode.png)

[![Continuous integration](https://github.com/stacklok/minder/actions/workflows/main.yml/badge.svg)](https://github.com/stacklok/minder/actions/workflows/main.yml) | [![Coverage Status](https://coveralls.io/repos/github/stacklok/minder/badge.svg?branch=main)](https://coveralls.io/github/stacklok/minder?branch=main) | [![License: Apache 2.0](https://img.shields.io/badge/License-Apache2.0-brightgreen.svg)](https://opensource.org/licenses/Apache-2.0) | [![SLSA 3](https://slsa.dev/images/gh-badge-level3.svg)](https://slsa.dev) | [![](https://dcbadge.vercel.app/api/server/RkzVuTp3WK?logo=discord&label=Discord&color=5865&style=flat)](https://discord.gg/RkzVuTp3WK)
---

[Installation](https://minder-docs.stacklok.dev/getting_started/install_cli) | [Documentation](https://minder-docs.stacklok.dev) | [Discussions](https://github.com/stacklok/minder/discussions) | [Releases](https://github.com/stacklok/minder/releases)
---

# What is Minder?

Minder by [Stacklok](https://stacklok.com/) is an open source platform that helps development teams and open source communities build more
secure software, and prove to others that what they’ve built is secure. Minder helps project owners proactively manage
their security posture by providing a set of checks and policies to minimize risk along the software supply chain,
and attest their security practices to downstream consumers.

Minder allows users to enroll repositories and define policy to ensure repositories and artifacts are configured
consistently and securely. Policies can be set to alert only or auto-remediate. Minder provides a predefined set of
rules and can also be configured to apply custom rules.

Minder can be deployed as a Helm chart and provides a CLI tool `minder`. Stacklok, the company behind Minder, also
provides a free-to-use hosted version of Minder (for public repositories only). Minder is designed to be extensible,
allowing users to integrate with their existing tooling and processes.

## Features

* **Repo configuration and security:** Simplify configuration and management of security settings and policies across repos.
* **Proactive security enforcement:** Continuously enforce best practice security configurations by setting granular policies to alert only or auto-remediate.
* **Artifact attestation:** Continuously verify that packages are signed to ensure they’re tamper-proof, using the open source project Sigstore.
* **Dependency management:** Manage dependency security posture by helping developers make better choices and enforcing controls. Minder is integrated with [Trusty by Stacklok](https://trustypkg.dev) to enable policy-driven dependency management based on the risk level of dependencies.

## Public instance

Your friends at Stacklok have set up a public instance of Minder that you can use for free. The Minder CLI tool
(`minder`) from our official releases is configured to use this instance by default. Follow Stacklok's Minder [Getting Started Guide](https://docs.stacklok.com/minder/getting_started/install_cli) to quickly try out Minder's features without having to build and deploy OSS Minder. 

Note that it's not possible to register private repositories. If you'd like to use Minder with private repositories,
feel free to [contact us](mailto:hello@stacklok.com)! We'd be thrilled to help you out.

---
# Getting Started (< 1 minute)

Getting up and running with Minder takes under a minute and is as easy as:

1. Installing Minder
2. Logging in to Minder
3. and running `minder quickstart` to create your first profile.

In just a few seconds, you will register your repositories and enable secret scanning protection for all of them! 🤯

<img src="https://github.com/stacklok/minder/assets/16540482/00646f28-2f48-43f2-bb2b-4a791782d7e3" width="80%"/>

## Installation

Choose your preferred method to install `minder`:

### MacOS (Homebrew)

Make sure you have [Homebrew](https://brew.sh/) installed.

```bash
brew install stacklok/tap/minder
```

### Windows (Winget)

Make sure you have [Winget](https://learn.microsoft.com/en-us/windows/package-manager/winget/) installed.

```bash
winget install stacklok.minder
```

### Download a release

Download the latest release from [minder/releases](https://github.com/stacklok/minder/releases).

### Build it from source

Build `minder` and `minder-server` from source by following the [build from source guide](#build-from-source).

## Logging in to Minder

To use `minder` with the [public instance](#public-instance) of Minder (`api.stacklok.com`), log in by running: 

```bash
minder auth login
```

Upon completion, you should see that the Minder Server is set to `api.stacklok.com`.


## Run Minder quickstart

The `quickstart` command guides you through creating your first profile in Minder, register your repositories, and enabling secret scanning protection for your repositories in seconds.

To do so, run:

```bash
minder quickstart
```

This will prompt you to enroll your provider, select the repositories you'd like, create the `secret_scanning`
rule type and create a profile which enables secret scanning for the selected repositories.

To see the status of your profile, run:

```bash
minder profile status list --profile quickstart-profile --detailed
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

# Development

This section describes how to build and run Minder from source.

## Build from source

### Prerequisites

You'd need the following tools available - [Go](https://golang.org/doc/install), [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/).

To build and run `minder-server`, you will also need [ko](https://ko.build/install/).

To run the test suite via `make test`, you will need [gotestfmt](https://github.com/GoTestTools/gotestfmt#installing) and [helm](https://github.com/helm/helm/releases).

### Clone the repository

```bash
git clone git@github.com:stacklok/minder.git
```

## Build 

Run the following to build `minder` and `minder-server` (binaries will be present at `./bin/`)

```bash
make build
```

To use `minder` with the public instance of Minder (`api.stacklok.com`), run:

```bash
minder auth login
```

Upon completion, you should see that the Minder Server is set to `api.stacklok.com`.

If you want to run `minder` against a local `minder-server` instance, proceed with the steps below.

#### Initial configuration

Create the initial configuration file for `minder`. You may do so by doing.

```bash
cp config/config.yaml.example config.yaml
```

Create the initial configuration file for `minder-server`. You may do so by doing.

```bash
cp config/server-config.yaml.example server-config.yaml
```

You'd also have to set up an OAuth2 application for `minder-server` to use.
Once completed, update the configuration file with the appropriate values.
See the documentation on how to do that - [Docs](https://minder-docs.stacklok.dev/run_minder_server/config_oauth).

#### Run `minder-server`

Start `minder-server` along with its dependant services (`keycloak` and `postgres`) by running:

```bash
make run-docker
```

#### Configure social login (GitHub)

`minder-server` uses Keycloak as an IAM. To log in, you'll need to set up a GitHub OAuth2 application and configure
Keycloak to use it.

Create an OAuth2 application for GitHub [here](https://github.com/settings/developers). Select
`New OAuth App` and fill in the details. The callback URL should be `http://localhost:8081/realms/stacklok/broker/github/endpoint`.
Create a new client secret for your OAuth2 client.

Using the `client_id` and `client_secret` you created above, enable GitHub login on Keycloak by running the following command:

```bash
make KC_GITHUB_CLIENT_ID=<client_id> KC_GITHUB_CLIENT_SECRET=<client_secret> github-login
```

#### Run minder

Ensure the `config.yaml` file is present in the current directory so `minder` can use it.

Run `minder` against your local instance of Minder (`localhost:8090`):

```bash
minder auth login
```

Upon completion, you should see that the Minder Server is set to `localhost:8090`.

By default, the `minder` CLI will point to the production Stacklok environment if a config file is not present, but [creating the `config.yaml` for running the server](#initial-configuration) will point the CLI at your local development environment.  If you explicitly want to use a different instance, you can set the `MINDER_CONFIG` environment variable to point to a particular configuration.  We have configurations for local development, the Stacklok production environment, and Stacklok staging environment (updated frequently) checked in to [the `config` directory](./config/).

### Development guidelines

You can find more detailed information about the development process in the [Developer Guide](https://minder-docs.stacklok.dev/developer_guide/get-hacking).

## Minder API

* REST API documentation - [Link](https://minder-docs.stacklok.dev/ref/api).

* Proto API documentation - [Link](https://minder-docs.stacklok.dev/ref/proto).

* Protobuf - [Link](https://github.com/stacklok/minder/blob/main/proto/minder/v1/minder.proto).

* OpenAPI/swagger spec (JSON) - [Link](https://github.com/stacklok/minder/blob/main/pkg/api/openapi/minder/v1/minder.swagger.json).

## Contributing

We welcome contributions to Minder. Please see our [Contributing](./CONTRIBUTING.md) guide for more information.

## Provenance

The Minder project follows the best practices for software supply chain security and transparency.

All released assets:

* Have a generated and verifiable SLSA Build Level 3 provenance. For more information, see the [SLSA website](https://slsa.dev).
* Have been signed and verified during release using the [Sigstore](https://sigstore.dev) project.
This ensures that
they are tamper-proof and can be verified by anyone.
* Have an SBOM archive generated and published along with the release.
This allows users to understand the dependencies of the project and their security posture.

## License

Minder is licensed under the [Apache 2.0 License](./LICENSE).
