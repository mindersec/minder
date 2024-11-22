---
title: minder datasource delete
---
## minder datasource delete

Delete a data source

### Synopsis

The datasource delete subcommand lets you delete a data source within Minder.

```
minder datasource delete [flags]
```

### Options

```
  -h, --help            help for delete
  -i, --id string       ID of the data source to delete
  -n, --name string     Name of the data source to delete
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
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder datasource](minder_datasource.md)	 - Manage data sources within a minder control plane

