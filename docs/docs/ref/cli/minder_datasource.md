---
title: minder datasource
---
## minder datasource

Manage data sources within a minder control plane

### Synopsis

The data source subcommand allows the management of data sources within Minder.

```
minder datasource [flags]
```

### Options

```
  -h, --help             help for datasource
  -j, --project string   ID of the project
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.custcodian.dev")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.custcodian.dev")
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder](minder.md)	 - Minder controls the hosted minder service
* [minder datasource apply](minder_datasource_apply.md)	 - Apply a data source
* [minder datasource create](minder_datasource_create.md)	 - Create a data source
* [minder datasource delete](minder_datasource_delete.md)	 - Delete a data source
* [minder datasource get](minder_datasource_get.md)	 - Get data source details
* [minder datasource list](minder_datasource_list.md)	 - List data sources

