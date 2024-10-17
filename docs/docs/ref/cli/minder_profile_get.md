---
title: minder profile get
---
## minder profile get

Get details for a profile

### Synopsis

The profile get subcommand lets you retrieve details for a profile within Minder.

```
minder profile get [flags]
```

### Options

```
  -h, --help            help for get
  -i, --id string       ID for the profile to query
  -n, --name string     Name for the profile to query
  -o, --output string   Output format (one of json,yaml,table) (default "table")
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
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder profile](minder_profile.md)	 - Manage profiles

