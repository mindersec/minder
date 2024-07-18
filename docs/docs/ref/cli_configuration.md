# Minder CLI configuration

The Minder CLI application is configured using a YAML file. The default location for the configuration file
is `$PWD/config.yaml`. You can specify a different location using the `--config` flag. If there's no configuration 
file at the specified location, the CLI application will use its default values.

## Prerequisites

* The `minder` CLI application
* A Stacklok account

## Configuration file example

Below is an example configuration file. The `grpc_server` section configures the gRPC server that the CLI
application will connect to. The `identity` section configures the issuer URL and client ID for the
Stacklok Identity service.

```yaml
---
# Minder CLI configuration
# gRPC server configuration
grpc_server:
  host: "127.0.0.1"
  port: 8090

identity:
  cli:
    issuer_url: http://localhost:8081
    client_id: minder-cli
---
```

## Handle multiple contexts using a configuration file 

The Minder CLI can be configured to use multiple contexts. A context is a set of configuration values that
are used to define a context, i.e. connect to a specific Minder server. For example, you may have a context for your local
development environment, a context for your staging environment, and a context for your production
environment. You can also specify things like the default `provider`, `project` or preferred format `output`
for each of those.

To create a new context, create a new configuration file and set the `MINDER_CONFIG` environment variable
to point to the config file.  For a single command, you can also set the path to the file through the `--config`
flag . For example, you can create your staging configuration in `config-staging.yaml` and use it as either:

```bash
export MINDER_CONFIG=./config-staging.yaml
minder auth login
# OR:
minder auth login --config ./config-staging.yaml
```
