---
title: minder profile status list
---
## minder profile status list

List profile status

### Synopsis

The profile status list subcommand lets you list profile status within Minder.

```
minder profile status list [flags]
```

### Options

```
  -d, --detailed          List all profile violations
  -h, --help              help for list
      --ruleName string   Filter profile status list by rule name
  -r, --ruleType string   Filter profile status list by rule type
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

