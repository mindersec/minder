Continuous integration | License 
 ----------------------|---------
 [![Continuous integration](https://github.com/stacklok/mediator/actions/workflows/main.yml/badge.svg)](https://github.com/stacklok/mediator/actions/workflows/main.yml) | [![License: Apache 2.0](https://img.shields.io/badge/License-Apache2.0-brightgreen.svg)](https://opensource.org/licenses/Apache-2.0)

# Mediator

Mediator is a platform to manage the security of your software supply chain.

It is currently in early development.

# Get Hacking (Development)

## Prerequisites

- [Go](https://golang.org/doc/install)
- [Docker](https://docs.docker.com/get-docker/) or..
- [Podman](https://podman.io/getting-started/installation)
- [Docker Compose](https://docs.docker.com/compose/install/) or..
- [Podman Compose](https://github.com/containers/podman-compose#installation)

Once you have these and have [cloned the repository](#clone-the-repository), you'll also need to [install the other tools](#install-tools) and make sure that `$HOME/go/bin` is in your `PATH`.

## Clone the repository

```bash
git clone git@github.com:stacklok/mediator.git
```

## Build the application

```bash
make build
```

## Initialize the configuration

Before running the makefile targets, you need to initialize the application's configuration file. You may do so by doing

```bash
cp config/config.yaml.example config.yaml
```

Alernatively, you may simply bootstrap the whole development environment, which includes initializing this file with:

```bash
make bootstrap
```

## Initialize the database

Both the `mediator` application and the tests need a Postgres database to be running.  For development use, the standard defaults should suffice:

```bash
docker-compose up -d postgres
make migrateup
```

## Run the application

You will need to [initialize the database](#initialize-the-database) before you can start the application.  Then run the application:

```bash
bin/mediator-server serve
```

Or direct from source

```bash
make run-server
```

The application will be available on `http://localhost:8080` and gRPC on `localhost:8090`.

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

To start with, you will need to run `cli auth login -u root -p P4ssw@rd` matching the password bootstrapped from the [database initialization](./database/migrations/000001_init.up.sql)

## APIs

The APIs are defined in protobuf [here](https://github.com/stacklok/mediator/blob/main/proto/mediator/v1/mediator.proto).

An OpenAPI / swagger spec is generated to [here](https://github.com/stacklok/mediator/blob/main/pkg/generated/openapi/proto/mediator/v1/mediator.swagger.json)

It can be accessed over gRPC or HTTP using [gprc-gateway](https://grpc-ecosystem.github.io/grpc-gateway/).

## How to generate protobuf stubs

We use [buf](https://buf.build/docs/) to generate the gRPC / HTTP stubs (both protobuf and openAPI). 

To build the stubs, run:

```bash
make gen
```

# Database migrations and tooling

Mediator uses [sqlc](https://sqlc.dev/) to generate Go code from SQL.

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

Mediator uses [mockgen](https://github.com/golang/mock) to generate mocks.

To generate the mocks, run:

```bash
mockgen -package mockdb -destination database/mock/store.go github.com/stacklok/mediator/pkg/db Store
```

# Configuration

Mediator uses [viper](https://github.com/spf13/viper) for configuration.

An example configuration file is `config/config.yaml.example`.

Most values should be quite self explanatory.

Before running the app, please copy the content of `config/config.yaml.example` into `$PWD/config.yaml` file, and modify to use your own settings.

# Initial setup / Getting started

## Login

First, login with the default password for the database:

```bash
go run ./cmd/cli/main.go auth login -u root -p P4ssw@rd
```

This will result in the following prompt:

```
You have been successfully logged in. Your access credentials saved to /var/home/jaosorior/.config/mediator/credentials.json
Remember that if that's your first login, you will need to update your password using the user update --password command
```

At this point, you should update the password:

```bash
go run ./cmd/cli/main.go user update -p f00b@r123 -c f00b@r123
```

And subsequently log in again with your new password

```bash
go run ./cmd/cli/main.go auth login -u root -p 'f00b@r123'
```

## Enroll provider

First, you'll need to enroll your first provider. Before doing this, make sure to set up a GitHub OAuth2 Application,
and fill in the appropriate settings in your `config.yaml` file.

You may create an OAuth2 application [here](https://github.com/settings/developers). Select
`New OAuth App` and fill in the details. The callback URL should be `http://localhost:8080/api/v1/auth/callback/github/cli`.
Create a new client secret and fill in the `client_id` and `client_secret` in your `config.yaml` file.

Once the Application is registered and the configuration is set, you can enroll the provider:

```bash
go run ./cmd/cli/main.go enroll provider -n github
```

This will take you through the OAuth2 flow and will result in the provider filling up the
repositories table with the repositories you have access to.

## Register repositories

Now that you've granted the GitHub app permissions to access your repositories, you can register them:

```bash
go run ./cmd/cli/main.go repo register -n github -g 1
```

You're probably wondering why you need to pass the `-g` flag. This is because the repositories are registered
under a group. This is to allow for multiple repositories to be registered under the same group. In the future,
you might create different groups for different purposes. For example, you might have a group for your personal
repositories, and another group for your work repositories.

Once you've registered the repositories, the mediator server will listen for events from GitHub and will
automatically create the necessary webhooks for you.
