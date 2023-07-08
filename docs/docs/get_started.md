---
id: get_started
title: Getting Started (Run the server)
sidebar_position: 2
slug: /get_started
displayed_sidebar: mediator
---

# Getting Started (Run the Server)

Mediator is platform, comprising of a controlplane, a CLI and a database.

The control plane runs two endpoints, a gRPC endpoint and a HTTP endpoint.

Mediator is controlled and managed via the CLI application `medic`.

PostgreSQL is used as the database.

There are two methods to get started with Mediator, either by downloading the
latest release, building from source or (quickest) using the provided `docker-compose`
file.

## Prerequisites

- [Go 1.20](https://golang.org/doc/install)
- [PostgreSQL](https://www.postgresql.org/download/)

## Download the latest release

[stub for when we cut a first release]

## Build from source

Alternatively, you can build from source.

### Clone the repository

```bash
git clone git@github.com:stacklok/mediator.git
```

### Build the application

```bash
make build
```

This will create two binaries, `bin/mediator-server` and `bin/medic`.

You may now copy these into a location on your path, or run them directly from the `bin` directory.

You will also need a configuration file. You can copy the example configuration file from `configs/config.yaml.example` to `~/.mediator.yaml`.

## Database creation

Mediator requires a PostgreSQL database to be running. You can install this locally, or use a container.

Should you install locally, you will need to set certain configuration options in your `~/.mediator.yaml` file, to reflect your local database configuration.

```yaml
database:
  dbhost: "localhost"
  dbport: 5432
  dbuser: postgres
  dbpass: postgres
  dbname: mediator
  sslmode: disable
```

### Using a container

A simple way to get started is to use the provided `docker-compose` file.

```bash
docker-compose up -d postgres
```

### Create the database

Once you have a running database, you can create the database using the `mediator-server` CLI tool or via the `make` command.

```bash
make migrateup
```

```bash
mediator-server migrate up
```

## Create encryption keys

Encryption keys are used to encrypt JWT tokens. You can create these using the `opensssl` CLI tool.

```bash
ssh-keygen -t rsa -b 2048 -m PEM -f access_token_rsa
ssh-keygen -t rsa -b 2048 -m PEM -f refresh_token_rsa
# For passwordless keys, run the following:
openssl rsa -in access_token_rsa -pubout -outform PEM -out access_token_rsa.pub
openssl rsa -in access_token_rsa -pubout -outform PEM -out access_token_rsa.pub
```

These keys should be placed in the `.ssh` directory, from where you will run the `mediator-server` binary. Alternatively, you can specify the location of the keys in the `./config.yaml` file.

```yaml
auth:
  access_token_private_key: "./.ssh/access_token_rsa"
  access_token_public_key: "./.ssh/access_token_rsa.pub"
  refresh_token_private_key: "./.ssh/refresh_token_rsa"
  refresh_token_public_key: "./.ssh/refresh_token_rsa.pub"
```

## Run the application

```bash
mediator-server serve
```

The application will be available on `http://localhost:8080` and gRPC on `localhost:8090`.