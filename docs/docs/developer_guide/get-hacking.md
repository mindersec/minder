---
title: Get hacking
sidebar_position: 1
---

# Get Hacking

## Run Minder
Follow the steps in the [Installing a Development version](./../run_minder_server/run_the_server.md) guide.

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

Users will then need to perform a migration

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
