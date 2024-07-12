---
title: minder history
---
## minder history

View evaluation history

### Synopsis

The history subcommands allows evaluation history to be viewed.

```
minder history [flags]
```

### Options

```
  -h, --help             help for history
  -o, --output string    Output format (one of json,yaml,table) (default "table")
  -j, --project string   ID of the project
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.stacklok.com")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")
```

### SEE ALSO

* [minder](minder.md)	 - Minder controls the hosted minder service
* [minder history list](minder_history_list.md)	 - List history

