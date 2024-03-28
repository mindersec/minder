---
title: minder ruletype
---
## minder ruletype

Manage rule types

### Synopsis

The ruletype subcommands allows the management of rule types within Minder.

```
minder ruletype [flags]
```

### Options

```
  -h, --help             help for ruletype
  -j, --project string   ID of the project
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.stacklok.com")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")
```

### SEE ALSO

* [minder](minder.md)	 - Minder controls the hosted minder service
* [minder ruletype apply](minder_ruletype_apply.md)	 - Apply a rule type
* [minder ruletype create](minder_ruletype_create.md)	 - Create a rule type
* [minder ruletype delete](minder_ruletype_delete.md)	 - Delete a rule type
* [minder ruletype get](minder_ruletype_get.md)	 - Get details for a rule type
* [minder ruletype list](minder_ruletype_list.md)	 - List rule types

