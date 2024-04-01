---
title: minder repo get
---
## minder repo get

Get repository details

### Synopsis

The repo get subcommand is used to get details for a registered repository within Minder.

```
minder repo get [flags]
```

### Options

```
  -h, --help            help for get
  -i, --id string       ID of the repo to query
  -n, --name string     Name of the repository (owner/name format)
  -o, --output string   Output format (one of json,yaml) (default "json")
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
```

### SEE ALSO

* [minder repo](minder_repo.md)	 - Manage repositories

