---
title: minder repo list
---
## minder repo list

List repositories

### Synopsis

The repo list subcommand is used to list registered repositories within Minder.

```
minder repo list [flags]
```

### Options

```
  -h, --help            help for list
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
  -p, --provider string          Name of the provider, i.e. github
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder repo](minder_repo.md)	 - Manage repositories

