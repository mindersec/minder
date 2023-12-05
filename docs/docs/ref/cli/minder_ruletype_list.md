---
title: minder ruletype list
---
## minder ruletype list

List rule types within a minder control plane

### Synopsis

The minder ruletype list subcommand lets you list rule type within a
minder control plane for an specific project.

```
minder ruletype list [flags]
```

### Options

```
  -h, --help              help for list
  -o, --output string     Output format (json, yaml or table) (default "table")
  -p, --provider string   Provider to list rule types for
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.stacklok.com")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-realm string    Identity server realm (default "stacklok")
      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")
```

### SEE ALSO

* [minder ruletype](minder_ruletype.md)	 - Manage rule types within a minder control plane

