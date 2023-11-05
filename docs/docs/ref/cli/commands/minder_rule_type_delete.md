---
title: minder rule type delete
---
## minder rule_type delete

Delete a rule type

### Synopsis

The minder rule type delete subcommand lets you delete rule types within a
minder control plane.

```
minder rule_type delete [flags]
```

### Options

```
  -a, --all               Warning: Deletes all rule types
  -h, --help              help for delete
  -i, --id string         ID of rule type to delete
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

* [minder rule_type](minder_rule_type.md)	 - Manage rule types within a minder control plane

