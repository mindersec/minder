---
title: minder repo delete
---
## minder repo delete

Delete a repository

### Synopsis

The repo delete subcommand is used to delete a registered repository within Minder.

```
minder repo delete [flags]
```

### Options

```
  -h, --help          help for delete
  -i, --id string     ID of the repo to delete
  -n, --name string   Name of the repository (owner/name format) to delete
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

