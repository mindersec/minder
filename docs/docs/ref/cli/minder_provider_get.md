---
title: minder provider get
---
## minder provider get

Get a given provider available in a specific project

### Synopsis

The minder provider get command gets a given provider available in a specific project.

```
minder provider get [flags]
```

### Options

```
  -h, --help            help for get
  -n, --name string     Name of the provider to get
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
  -p, --provider class           DEPRECATED - use class flag of `enroll` instead
```

### SEE ALSO

* [minder provider](minder_provider.md)	 - Manage providers within a minder control plane

