---
title: minder entity delete
---
## minder entity delete

Delete an entity

### Synopsis

The entity delete subcommand is used to delete an entity instance within Minder.

```
minder entity delete [flags]
```

### Options

```
  -h, --help        help for delete
  -i, --id string   ID of the entity to delete
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
  -p, --provider string          Name of the provider, i.e. github
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder entity](minder_entity.md)	 - Manage entities within a Minder project

