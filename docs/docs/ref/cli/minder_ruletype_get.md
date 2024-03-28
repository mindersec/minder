---
title: minder ruletype get
---
## minder ruletype get

Get details for a rule type

### Synopsis

The ruletype get subcommand lets you retrieve details for a rule type within Minder.

```
minder ruletype get [flags]
```

### Options

```
  -h, --help            help for get
  -i, --id string       ID for the rule type to query
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
```

### SEE ALSO

* [minder ruletype](minder_ruletype.md)	 - Manage rule types

