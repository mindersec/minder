![minder logo](./docs/docs/images/Minder_darkMode.png)

[![Continuous integration](https://github.com/stacklok/minder/actions/workflows/main.yml/badge.svg)](https://github.com/stacklok/minder/actions/workflows/main.yml) | [![License: Apache 2.0](https://img.shields.io/badge/License-Apache2.0-brightgreen.svg)](https://opensource.org/licenses/Apache-2.0) | [![SLSA 3](https://slsa.dev/images/gh-badge-level3.svg)](https://slsa.dev)
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

Minder can be deployed as a Helm chart and provides a CLI tool ‘minder’. Stacklok, the company behind Minder, also
provides a free-to-use hosted version of Minder (for public repositories only). Minder is designed to be extensible,
allowing users to integrate with their existing tooling and processes.

## Features

* **Repo configuration and security:** Simplify configuration and management of security settings and policies across repos.
* **Proactive security enforcement:** Continuously enforce best practice security configurations by setting granular policies to alert only or auto-remediate.
* **Artifact attestation:** Continuously verify that packages are signed to ensure they’re tamper-proof, using the open source project Sigstore.
* **Dependency management:** Manage dependency security posture by helping developers make better choices and enforcing controls. Minder is integrated with [Trusty by Stacklok](https://trustypkg.dev) to enable policy-driven dependency management based on the risk level of dependencies.

---
## Stacklok Instance

Your friends at Stacklok have set up a public instance of Minder that you can use for free. The Minder CLI tool
(`minder`) from our official releases is configured to use this instance by default. You can also use the public
instance by running `minder auth login` and following the prompts.

```bash
minder auth login --grpc-host api.stacklok.com --identity-url https://auth.stacklok.com
```

Note that it's not possible to register private repositories. If you'd like to use Minder with private repositories,
feel free to [contact us](mailto:hello@stacklok.com)! We'd be thrilled to help you out.

---
# Getting Started

## Installation

You can install `minder` using one of the following methods:

### MacOS (Homebrew)

```bash
brew install stacklok/tap/minder
```

### Windows (Winget)

```bash
winget install stacklok.minder
```

### Releases

Download the latest release from - [minder/releases](https://github.com/stacklok/minder/releases).

### Build it from source

Build `minder` and `minder-server` from source by following - [#build-from-source](#build-from-source).

## Run minder

To use `minder` with the public instance of Minder (`api.stacklok.com`), run: 

```bash
minder auth login
```

Upon completion, you should see that the Minder Server is set to `api.stacklok.com`.

## Enroll a repository provider

Minder supports GitHub as a provider to enroll repositories. To enroll your provider, run:

```bash
minder provider enroll --provider github
```

A browser session will open, and you will be prompted to login to your GitHub.
Once you have granted Minder access, you will be redirected back, and the user will be enrolled.
The minder CLI application will report the session is complete.

## Register a repository

Now that you've granted the GitHub app permissions to access your repositories, you can register them:

```bash
minder repo register --provider github
```

Once you've registered the repositories, the Minder server will listen for events from GitHub and will
automatically create the necessary webhooks for you.

Now you can run `minder` commands against the public instance of Minder where you can manage your registered repositories
and create custom profiles that would help ensure your repositories are configured consistently and securely.

For more information about `minder`, see:
* `minder` CLI commands - [Docs](https://minder-docs.stacklok.dev/ref/cli/minder).
* Minder documentation - [Docs](https://minder-docs.stacklok.dev).

# Development

## Build from source

### Prerequisites

You'd need the following tools available - [Go](https://golang.org/doc/install), [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/).

### Clone the repository

```bash
git clone git@github.com:stacklok/minder.git
```

## Build 

Run the following to build `minder` and `minder-server`(binaries will be present at `./bin/`)

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

Create the initial configuration file for `minder` and `minder-server`. You may do so by doing.

```bash
cp config/config.yaml.example config.yaml
```

You'd also have to set up an OAuth2 application for `minder-server` to use.
Once completed, update the configuration file with the appropriate values.
See the documentation on how to do that - [Docs](https://minder-docs.stacklok.dev/run_minder_server/config_oauth).

#### Run `minder-server`

Start `minder-server` along with its dependant services (`keycloak` and `postgres`) by running:

```bash
KO_DOCKER_REPO=ko.local make run-docker
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
