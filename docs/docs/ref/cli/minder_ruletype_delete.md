---
title: minder ruletype delete
---
## minder ruletype delete

Delete a rule type

### Synopsis

The ruletype delete subcommand lets you delete rule types within Minder.

```
minder ruletype delete [flags]
```

### Options

```
  -a, --all         Warning: Deletes all rule types
  -h, --help        help for delete
  -i, --id string   ID of rule type to delete
  -y, --yes         Bypass yes/no prompt when deleting all rule types
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

