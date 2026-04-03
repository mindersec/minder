---
title: minder repo
---
## minder repo

Manage repositories within a Minder project

### Synopsis

Manage repositories within a Minder project.

This command allows you to list, add, and manage repositories
connected to Minder for security analysis and policy enforcement.

```
minder repo [flags]
```

### Examples

```

  # List repositories
    minder repo list

  # Register a repository
    minder repo register --name my-repo --provider github

  # Delete a repository
    minder repo delete --name my-repo

```

### Options

```
  -h, --help              help for repo
  -j, --project string    ID of the project
  -p, --provider string   Name of the provider, i.e. github
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.custcodian.dev")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.custcodian.dev")
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder](minder.md)	 - Minder controls the hosted minder service
* [minder repo delete](minder_repo_delete.md)	 - Delete a repository
* [minder repo get](minder_repo_get.md)	 - Get repository details
* [minder repo list](minder_repo_list.md)	 - List repositories
* [minder repo reconcile](minder_repo_reconcile.md)	 - Reconcile (Sync) a repository with Minder.
* [minder repo register](minder_repo_register.md)	 - Register a repository

