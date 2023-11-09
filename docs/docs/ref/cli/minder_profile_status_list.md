---
title: minder profile status list
---
## minder profile_status list

List profile status within a minder control plane

### Synopsis

The minder profile_status list subcommand lets you list profile status within a
minder control plane for an specific provider/project or profile id.

```
minder profile_status list [flags]
```

### Options

```
  -d, --detailed          List all profile violations
  -h, --help              help for list
  -o, --output string     Output format (json, yaml or table) (default "table")
  -i, --profile string    Profile name to list profile status for
  -g, --project string    Project ID to list profile status for
  -p, --provider string   Provider to list profile status for (default "github")
  -r, --rule string       Filter profile status list by rule
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

