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

# Configuration

Mediator uses [viper](https://github.com/spf13/viper) for configuration.

An example configuration file is `config/config.yaml.example`.

Most values should be quite self explanatory.

Before running the app, please copy the content of `config/config.yaml.example` into `$PWD/config.yaml` file, and modify to use your own settings.

## Configure OAuth2

Mediator can use OAuth2 to authenticate users, support for Google and GitHub is
currently implemented.

To enable OAuth2, you need to configure accounts in the relevant provider.

### Configure GitHub OAuth2

To configure GitHub OAuth2, you need to create a new OAuth2 application in your
GitHub account. 

Head to the [GitHub OAuth2 applications page](https://github.com/settings/applications/new)

Give your application a name and set the homepage URL to `http://localhost:8080`.

The callback URL should be set to:

`http://localhost:8080/api/v1/auth/callback/github`.

![GitHub OAuth2 application](images/new-github-oauth2.png)

On the next page, you will see your client ID and you can generate a client secret.

Place these values in the `config.yaml` file:

```bash
github:
  client_id: "client_id"
  client_secret: "client_secret"
  redirect_uri: "http://localhost:8080/api/v1/auth/callback/github"
```

### Configure Google OAuth2

To configure Google OAuth2, you need to create a new OAuth2 application in your
Google account. 

Follow the official [Google OAuth2 documentation](https://developers.google.com/identity/protocols/oauth2/web-server#creatingcred).

The callback URL should be set to:

`http://localhost:8080/api/v1/auth/callback/google`.

Place these values in the `config.yaml` file:

```bash
github:
  client_id: "client_id"
  client_secret: "client_secret"
  redirect_uri: "http://localhost:8080/api/v1/auth/callback/google"
```

