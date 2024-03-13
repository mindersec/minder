---
title: Get Hacking
sidebar_position: 1
---

# Get Hacking

## Prerequisites

- [Go](https://golang.org/doc/install)
- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)

## Clone the repository

```bash
git clone git@github.com:stacklok/minder.git
```

## Build the application

```bash
make build
```

## Install tools

You may bootstrap the whole development environment, which includes initializing the `config.yaml` and `server-config.yaml`
files with:

```bash
make bootstrap
```
This also installs the required tools for running different make targets.

Note that if you intend to run minder outside `docker compose`, you should
change the Keycloak and OpenFGA URLs in `server-config.yaml` to refer to
localhost instead of the `docker-compose.yaml` names. There are comments inside the
config file which explain what needs to be changed.

## Start dependencies

Note that the application requires a database to be running. This can be achieved
using docker compose:

```bash
services="postgres keycloak migrate openfga" make run-docker
```

## Set up a Keycloak user

You have two options here: setting up a GitHub app (possibly the same one you
use for Minder enrollment), or using username / password.

### Username / password Keycloak user

Assuming that you've run `make run-docker`, you can run:

```bash
make password-login
```

to create a `testuser` Keycloak user with the password `tester`.  (You can create more users either through the KeyCloak UI or by modifying the command in [./mk/identity.mk](https://github.com/stacklok/minder/blob/main/.mk/identity.mk).)  This is purely intended as a convenience method, and is fairly fragile.

### GitHub App

[Create an OAuth2 application for GitHub](../run_minder_server/config_oauth.md).
Select `New OAuth App` and fill in the details.

Create a new client secret for your OAuth2 client.

Using the client ID and client secret you created above, enable GitHub login on Keycloak by running the following command:
```bash
make KC_GITHUB_CLIENT_ID=<client_id> KC_GITHUB_CLIENT_SECRET=<client_secret> github-login
```

## Run the application

Then run the application

```bash
bin/minder-server serve
```

Or direct from source

```bash
go run cmd/server/main.go serve
```

The application will be available on `https://localhost:8080` and gRPC on `https://localhost:8090`.

## Run the tests

```bash
make test
```

## CLI

The CLI is available in the `cmd/cli` directory.  You can also use the pre-built `minder` CLI with your new application; you'll need to set the `--grpc-host localhost --grpc-port 8090` arguments in either case.

```bash
go run cmd/cli/main.go --help
```

## APIs

The APIs are defined in protobuf [here](https://github.com/stacklok/minder/blob/main/proto/minder/v1/minder.proto).

An OpenAPI / swagger spec is generated to [here](https://github.com/stacklok/minder/blob/main/pkg/api/openapi/proto/minder/v1/minder.swagger.json)

It can be accessed over gRPC or HTTP using [gprc-gateway](https://grpc-ecosystem.github.io/grpc-gateway/).

## How to generate protobuf stubs

We use [buf](https://buf.build/docs/) to generate the gRPC / HTTP stubs (both protobuf and openAPI).

To build the stubs, run:

```bash
make clean-gen
make gen
```

## Database migrations and tooling

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

## Viper configuration

Minder uses [viper](https://github.com/spf13/viper) for configuration.

An example CLI configuration file is `config/config.yaml.example`.

An example server configuration file is `config/server-config.yaml.example`.

Most values should be quite self-explanatory.

Before running the app, please copy the content of `config/config.yaml.example` into `$PWD/config.yaml` file,
and `config/server-config.yaml.example` into `$PWD/server-config.yaml` file, and modify to use your own settings.
