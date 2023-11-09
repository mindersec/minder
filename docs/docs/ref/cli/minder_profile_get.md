---
title: minder profile get
---
## minder profile get

Get details for a profile within a minder control plane

### Synopsis

The minder profile get subcommand lets you retrieve details for a profile within a
minder control plane.

```
minder profile get [flags]
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

* [minder profile](minder_profile.md)	 - Manage profiles within a minder control plane

