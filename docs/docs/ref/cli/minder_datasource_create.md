---
title: minder datasource create
---
## minder datasource create

Create a data source

### Synopsis

The datasource create subcommand lets you create new data sources for a project within Minder.

```
minder datasource create [flags]
```

### Options

```
  -f, --file stringArray   Path to the YAML defining the data source (or - for stdin). Can be specified multiple times. Can be a directory.
  -h, --help               help for create
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.custcodian.dev")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.custcodian.dev")
  -j, --project string           ID of the project
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder datasource](minder_datasource.md)	 - Manage data sources within a minder control plane

