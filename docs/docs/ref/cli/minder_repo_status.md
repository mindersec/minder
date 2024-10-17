---
title: minder repo status
---
## minder repo status

Get repository evaluation status

### Synopsis

The repo status subcommand is used to get the evaluation status for a registered repository within Minder.

```
minder repo status [flags]
```

### Options

```
  -e, --entity string      Entity ID to get evaluation status for
  -h, --help               help for status
  -l, --labels string      Query by labels
  -o, --output string      Output format (one of json,yaml) (default "json")
  -n, --profile string     Query by a profile
  -r, --ruletypes string   Query by ruletypes, i.e ruletypes=rule1,rule2
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
  -p, --provider string          Name of the provider, i.e. github (default "github")
```

### SEE ALSO

* [minder repo](minder_repo.md)	 - Manage repositories

