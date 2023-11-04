Continuous integration | License 
 ----------------------|---------
 [![Continuous integration](https://github.com/stacklok/minder/actions/workflows/main.yml/badge.svg)](https://github.com/stacklok/minder/actions/workflows/main.yml) | [![License: Apache 2.0](https://img.shields.io/badge/License-Apache2.0-brightgreen.svg)](https://opensource.org/licenses/Apache-2.0)

# Minder

Minder is a platform to manage the security of your software supply chain. It helps teams automate and enforce secure supply chain best practices across the software development lifecycle.  

It is currently in early and active development and is considered experimental.

# Get Hacking (Development)

## Prerequisites

- [Go](https://golang.org/doc/install)
- [Docker](https://docs.docker.com/get-docker/) or..
- [Podman](https://podman.io/getting-started/installation)
- [Docker Compose](https://docs.docker.com/compose/install/) or..

Once you have these and have [cloned the repository](#clone-the-repository), you'll also need to [install the other tools](#install-tools) and make sure that `$HOME/go/bin` is in your `PATH`.

## Clone the repository

```bash
git clone git@github.com:stacklok/minder.git
```

## Build the application

```bash
make build
```

## Initialize the configuration

Before running the makefile targets, initialize the application's configuration file. You may do so by doing.

```bash
cp config/config.yaml.example config.yaml
```

Alternatively, you may bootstrap the whole development environment, which includes initializing this file with:

```bash
make bootstrap
```

## Initialize the database

Both the `minder` application and the tests need a Postgres database to be running.  For development use, the standard defaults should suffice:

```bash
docker-compose up -d postgres
make migrateup
```

## Start the identity provider (Keycloak)

To login, we rely on an identity provider that stores the usernames and passwords.

```bash
docker-compose up -d keycloak
```

## Run the application

You must [initialize the database](#initialize-the-database) before starting the application.  Then run the application:

```bash
bin/minder-server serve
```

Or direct from source

```bash
make run-server
```

The application will be available on `http://localhost:8080` and gRPC on `localhost:8090`.

## Running the server under Compose:

**NOTE: the command will be `docker-compose`.**  You'll need to install the [`ko`](https://ko.build/install/) tool do the build and run.

```bash
# The repo to push to; "ko.local" is a special string meaning your local Docker repo
KO_DOCKER_REPO=ko.local
# ko adds YAML document separators at the end of each document, which docker-compose doesn't like
docker-compose -f <(ko resolve -f docker-compose.yaml | sed 's/^---$//') up
```

## Run the tests

Note that you need to have [started the database and loaded the schema](#initialize-the-database) before running the tests:

```bash
make test
```

You can alse use `make cover` to check coverage.

## Install tools

```bash
make bootstrap
```

## CLI

The CLI is available in the `cmd/cli` directory.

```bash
go run cmd/cli/main.go --help 
```

To start with, you will need to run `cli auth login` using `root:root` as the credentials.
This will open a browser window with the identity provider login page.

## APIs

API Doc [here](https://minder-docs.stacklok.dev/api)

The APIs are defined in protobuf [here](https://github.com/stacklok/minder/blob/main/proto/minder/v1/minder.proto).

An OpenAPI / swagger spec is generated to [JSON](https://github.com/stacklok/minder/blob/main/pkg/api/openapi/minder/v1/minder.swagger.json) 

It can be accessed over gRPC or HTTP using [gprc-gateway](https://grpc-ecosystem.github.io/grpc-gateway/).

## How to generate protobuf stubs

We use [buf](https://buf.build/docs/) to generate the gRPC / HTTP stubs (both protobuf and openAPI). 

To build the stubs, run:

```bash
make gen
```

# Database migrations and tooling

Minder uses [sqlc](https://sqlc.dev/) to generate Go code from SQL.

The main configuration file is `sqlc.yaml`.

To make changes to the database schema, create a new migration file in the
`database/migrations` directory.

Add any queries to the `database/queries/sqlc.sql` file.

To generate the Go code, run:

```bash
make sqlc
```

Users will then need to peform a migration

```bash
make migrateup 
``` 

```bash
make migratedown
```

# Mock

Minder uses [mockgen](https://github.com/golang/mock) to generate mocks.

To generate the mocks, run:

```bash
mockgen -package mockdb -destination database/mock/store.go github.com/stacklok/minder/internal/db Store
```
and
```bash
mockgen -package auth -destination internal/auth/mock/jwtauth.go github.com/stacklok/minder/internal/auth JwtValidator,KeySetFetcher
```

# Configuration

Minder uses [viper](https://github.com/spf13/viper) for configuration.

An example configuration file is `config/config.yaml.example`.

Most values should be quite self-explanatory.

Before running the app, please copy the content of `config/config.yaml.example` into `$PWD/config.yaml` file, and modify to use your own settings.

## Social login configuration
First, create an OAuth2 application for GitHub [here](https://github.com/settings/developers). Select
`New OAuth App` and fill in the details. The callback URL should be `http://localhost:8081/realms/stacklok/broker/github/endpoint`.
Create a new client secret for your OAuth2 client.

Using the client ID and client secret you created above, enable GitHub login on Keycloak by running the following command:
```bash
make KC_GITHUB_CLIENT_ID=<client_id> KC_GITHUB_CLIENT_SECRET=<client_secret> github-login
```

# Initial setup / Getting started

## Login

First, login with the default credentials:

```bash
go run ./cmd/cli/main.go auth login
```

This will open a browser window with the identity provider login page.
Enter the credentials `root:root`.
You will immediately be prompted to change your password.
Upon successful authentication you can close your browser.

You will see the following prompt in your terminal:

```
You have been successfully logged in. Your access credentials saved to /var/home/jaosorior/.config/minder/credentials.json
```

## Enroll provider

First, you'll need to enroll your first provider. Before doing this, make sure to set up a GitHub OAuth2 Application,
and fill in the appropriate settings in your `config.yaml` file.

You may create an OAuth2 application [here](https://github.com/settings/developers). Select
`New OAuth App` and fill in the details. The callback URL should be `http://localhost:8080/api/v1/auth/callback/github`.
Create a new client secret and fill in the `client_id` and `client_secret` in your `config.yaml` file.

Once the Application is registered and the configuration is set, you can enroll the provider:

```bash
go run ./cmd/cli/main.go provider enroll -n github
```

This will take you through the OAuth2 flow and will result in the provider filling up the
repositories table with the repositories you have access to.

## Register repositories

Now that you've granted the GitHub app permissions to access your repositories, you can register them:

```bash
go run ./cmd/cli/main.go repo register -n github
```

Once you've registered the repositories, the Minder server will listen for events from GitHub and will
automatically create the necessary webhooks for you.
