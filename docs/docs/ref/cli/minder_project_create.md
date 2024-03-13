---
title: minder project create
---
## minder project create

Create a sub-project within a minder control plane

### Synopsis

The list command lists the projects available to you within a minder control plane.

```
minder project create [flags]
```

### Options

```
  -h, --help             help for create
  -n, --name string      The name of the project to create
  -o, --output string    Output format (one of json,yaml,table) (default "table")
  -j, --project string   The project to create the sub-project within
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

