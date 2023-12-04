---
title: Run the Server
sidebar_position: 10
---

# Run a minder server

Minder is platform, comprising of a controlplane, a CLI, a database and an identity provider.

The control plane runs two endpoints, a gRPC endpoint and a HTTP endpoint.

Minder is controlled and managed via the CLI application `minder`.

PostgreSQL is used as the database.

Keycloak is used as the identity provider.

There are two methods to get started with Minder, either by downloading the
latest release, building from source or (quickest) using the provided `docker-compose`
file.

## Prerequisites

- [Go 1.20](https://golang.org/doc/install)
- [PostgreSQL](https://www.postgresql.org/download/)
- [Keycloak](https://www.keycloak.org/guides)

## Download the latest release

[stub for when we cut a first release]

## Build from source

Alternatively, you can build from source.

### Clone the repository

```bash
git clone git@github.com:stacklok/minder.git
```

### Build the application

```bash
make build
```

This will create two binaries, `bin/minder-server` and `bin/minder`.

You may now copy these into a location on your path, or run them directly from the `bin` directory.

You will also need a configuration file. You can copy the example configuration file from `configs/config.yaml.example` to `$(PWD)/config.yaml`.

If you prefer to use a different file name or location, you can specify this using the `--config` 
flag, e.g. `minder-server --config /file/path/config.yaml serve` when you later run the application.

## Database creation

Minder requires a PostgreSQL database to be running. You can install this locally, or use a container.

Should you install locally, you will need to set certain configuration options in your `config.yaml` file, to reflect your local database configuration.

```yaml
database:
  dbhost: "localhost"
  dbport: 5432
  dbuser: postgres
  dbpass: postgres
  dbname: minder
  sslmode: disable
```

### Using a container

A simple way to get started is to use the provided `docker-compose` file.

```bash
docker-compose up -d postgres
```

### Create the database

Once you have a running database, you can create the database using the `minder-server` CLI tool or via the `make` command.

```bash
make migrateup
```

or:

```bash
minder-server migrate up
```

## Identity Provider
Minder requires a Keycloak instance to be running. You can install this locally, or use a container.

Should you install locally, you will need to configure the client on Keycloak.
You will need the following:
- A Keycloak realm named "stacklok" with event saving turned on for the "Delete account" event.
- A registered public client with the redirect URI `http://localhost/*`. This is used for the minder CLI.
- A registered confidential client with a service account that can manage users and view events. This is used for the minder server.

You will also need to set certain configuration options in your `config.yaml` file, to reflect your local Keycloak configuration.
```yaml
identity:
  cli:
    issuer_url: http://localhost:8081
    client_id: minder-cli
  server:
    issuer_url: http://localhost:8081
    client_id: minder-server
    client_secret: secret
```

### Using a container

A simple way to get started is to use the provided `docker-compose` file.

```bash
docker-compose up -d keycloak
```

### Social login
Once you have a Keycloak instance running locally, you can set up GitHub authentication.

#### Create a GitHub OAuth Application

1. Navigate to [GitHub Developer Settings](https://github.com/settings/profile)
2. Select "Developer Settings" from the left hand menu
3. Select "OAuth Apps" from the left hand menu
4. Select "New OAuth App"
5. Enter the following details:
    - Application Name: `Stacklok Identity Provider`
    - Homepage URL: `http://localhost:8081` or the URL you specified as the `issuer_url` in your `config.yaml`
    - Authorization callback URL: `http://localhost:8081/realms/stacklok/broker/github/endpoint`
6. Select "Register Application"
7. Generate a client secret

![github oauth2 page](./images/github-settings-application.png)

#### Enable GitHub login

Using the client ID and client secret you created above, enable GitHub login your local Keycloak instance by running the 
following command:
```bash
make KC_GITHUB_CLIENT_ID=<client_id> KC_GITHUB_CLIENT_SECRET=<client_secret> github-login
```

## Create token key passphrase

Create a token key passphrase that is used when storing the provider's token in the database. 

The default configuration expects these keys to be in a directory named `.ssh`, relative to where you run the `minder-server` binary.
Start by creating the `.ssh` directory.

```bash
mkdir .ssh
```

You can create the passphrase using the `openssl` CLI tool.

```bash
openssl rand -base64 32 > .ssh/token_key_passphrase
```

If your key lives in a directory other than `.ssh`, you can specify the location of the key in the `config.yaml` file.

```yaml
auth:
   token_key: "./.ssh/token_key_passphrase"
```

## Run the application

```bash
minder-server serve
```

The application will be available on `http://localhost:8080` and gRPC on `localhost:8090`.