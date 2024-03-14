---
title: minder repo
---
## minder repo

Manage repositories

### Synopsis

The repo commands allow the management of repositories within Minder.

```
minder repo [flags]
```

### Options

```
  -h, --help              help for repo
  -j, --project string    ID of the project
  -p, --provider string   Name of the provider, i.e. github (default "github")
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
* [minder repo delete](minder_repo_delete.md)	 - Delete a repository
* [minder repo get](minder_repo_get.md)	 - Get repository details
* [minder repo list](minder_repo_list.md)	 - List repositories
* [minder repo reconcile](minder_repo_reconcile.md)	 - Reconcile (Sync) a repository with Minder.
* [minder repo register](minder_repo_register.md)	 - Register a repository

