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

## Run the application

Note that the application requires a database to be running. This can be achieved
using docker-compose:

```bash
services="postgres keycloak migrateup" make run-docker
```

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

## Install tools

You may bootstrap the whole development environment, which includes initializing the `config.yaml` file with:

```bash
make bootstrap
```

## CLI

The CLI is available in the `cmd/cli` directory.

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

# Viper configuration

Minder uses [viper](https://github.com/spf13/viper) for configuration.

An example configuration file is `config/config.yaml.example`.

Most values should be quite self-explanatory.

Before running the app, please copy the content of `config/config.yaml.example` into `$PWD/config.yaml` file,
and modify to use your own settings.

# Keycloak configuration for social login (GitHub)
Create an OAuth2 application for GitHub [here](https://github.com/settings/developers). Select
`New OAuth App` and fill in the details. The callback URL should be `http://localhost:8081/realms/stacklok/broker/github/endpoint`.
Create a new client secret for your OAuth2 client.

Using the client ID and client secret you created above, enable GitHub login on Keycloak by running the following command:
```bash
make KC_GITHUB_CLIENT_ID=<client_id> KC_GITHUB_CLIENT_SECRET=<client_secret> github-login
```
