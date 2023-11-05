---
title: minder profile status get
---
## minder profile_status get

Get profile status within a minder control plane

### Synopsis

The minder profile_status get subcommand lets you get profile status within a
minder control plane for an specific provider/project or profile id, entity type and entity id.

```
minder profile_status get [flags]
```

### Options

```
  -e, --entity string        Entity ID to get profile status for
  -t, --entity-type string   the entity type to get profile status for (one of artifact,build_environment,repository)
  -h, --help                 help for get
  -o, --output string        Output format (json, yaml or table) (default "table")
  -i, --profile string       Profile name to get profile status for
  -g, --project string       Project ID to get profile status for
  -p, --provider string      Provider to get profile status for (default "github")
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

* [minder profile_status](minder_profile_status.md)	 - Manage profile status within a minder control plane

