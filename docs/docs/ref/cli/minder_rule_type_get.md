---
title: minder rule type get
---
## minder rule_type get

Get details for a rule type within a minder control plane

### Synopsis

The minder rule_type get subcommand lets you retrieve details for a rule type within a
minder control plane.

```
minder rule_type get [flags]
```

### Options

```
  -h, --help              help for get
  -i, --id string         ID for the profile to query
  -o, --output string     Output format (json, yaml or table) (default "table")
  -p, --provider string   Provider for the profile (default "github")
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

* [minder rule_type](minder_rule_type.md)	 - Manage rule types within a minder control plane

