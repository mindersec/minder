---
title: minder project list
---
## minder project list

List the projects available to you within a minder control plane

### Synopsis

The list command lists the projects available to you within a minder control plane.

```
minder project list [flags]
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
```

### SEE ALSO

* [minder project](minder_project.md)	 - Manage project within a minder control plane

