---
id: get_hacking
title: Developer Guide
sidebar_position: 5
slug: /get_hacking
displayed_sidebar: mediator
---

# Get Hacking (Development)

## Prerequisites

- [Go](https://golang.org/doc/install)
- [Docker](https://docs.docker.com/get-docker/) or..
- [Podman](https://podman.io/getting-started/installation)
- [Docker Compose](https://docs.docker.com/compose/install/) or..
- [Podman Compose](https://github.com/containers/podman-compose#installation)

## Clone the repository

```bash
git clone git@github.com:stacklok/mediator.git
```

## Build the application

```bash
make build
```

## Run the application

Note that the application requires a database to be running. This can be achieved
using docker-compose:

```bash
docker-compose up -d postgres
```

Then run the application

```bash
bin/mediator-server serve
```

Or direct from source

```bash
go run cmd/server/main.go serve
```

The application will be available on `http://localhost:8080` and gRPC on `localhost:8090`.

## Run the tests

```bash
make test
```

## Install tools

```bash
make bootstrap
```

## CLI

The CLI is available in the `cmd/cli` directory.

```bash
go run cmd/cli/main.go --help
```

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
