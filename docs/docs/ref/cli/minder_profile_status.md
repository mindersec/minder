---
title: minder profile status
---
## minder profile status

Manage profile status

### Synopsis

The profile status subcommand allows management of profile status within Minder.

```
minder profile status [flags]
```

### Options

```
  -h, --help            help for status
  -o, --output string   Output format (one of json,yaml,table) (default "table")
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
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder profile](minder_profile.md)	 - Manage profiles
* [minder profile status get](minder_profile_status_get.md)	 - Get profile status
* [minder profile status list](minder_profile_status_list.md)	 - List profile status

