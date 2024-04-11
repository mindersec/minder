---
title: minder provider delete
---
## minder provider delete

Delete a given provider available in a specific project

### Synopsis

The minder provider delete command deletes a given provider available in a specific project.

```
minder provider delete [flags]
```

### Options

```
  -h, --help          help for delete
  -i, --id string     ID of the provider to delete
  -n, --name string   Name of the provider to delete
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
```

### SEE ALSO

* [minder provider](minder_provider.md)	 - Manage providers within a minder control plane

