---
title: minder profile list
---
## minder profile list

List profiles

### Synopsis

The profile list subcommand lets you list profiles within Minder.

```
minder profile list [flags]
```

### Options

```
  -h, --help            help for list
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

