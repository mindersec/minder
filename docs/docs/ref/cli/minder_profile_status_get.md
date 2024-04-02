---
title: minder profile status get
---
## minder profile status get

Get profile status

### Synopsis

The profile status get subcommand lets you get profile status within Minder.

```
minder profile status get [flags]
```

### Options

```
  -e, --entity string        Entity ID to get profile status for
  -t, --entity-type string   the entity type to get profile status for (one of artifact, build_environment, repository)
  -h, --help                 help for get
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.stacklok.com")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")
  -n, --name string              Profile name to get profile status for
  -o, --output string            Output format (one of json,yaml,table) (default "table")
  -j, --project string           ID of the project
```

### SEE ALSO

* [minder profile status](minder_profile_status.md)	 - Manage profile status

