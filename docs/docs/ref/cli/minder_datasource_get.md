---
title: minder datasource get
---
## minder datasource get

Get data source details

### Synopsis

The datasource get subcommand lets you retrieve details for a data source within Minder.

```
minder datasource get [flags]
```

### Options

```
  -h, --help            help for get
  -i, --id string       ID of the data source to get info from
  -n, --name string     Name of the data source to get info from
  -o, --output string   Output format (one of json,yaml,table) (default "table")
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.stacklok.com")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")
  -j, --project string           ID of the project
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder datasource](minder_datasource.md)	 - Manage data sources within a minder control plane

