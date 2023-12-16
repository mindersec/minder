---
title: minder quickstart
---
## minder quickstart

Quickstart minder

### Synopsis

The quickstart command provide the means to quickly get started with minder

```
minder quickstart [flags]
```

### Options

```
  -h, --help              help for quickstart
  -o, --owner string      Owner to filter on for provider resources
  -r, --project string    Project to create the quickstart profile in
  -p, --provider string   Name of the provider (default "github")
  -t, --token string      Personal Access Token (PAT) to use for enrollment
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

