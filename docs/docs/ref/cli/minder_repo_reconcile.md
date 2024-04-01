---
title: minder repo reconcile
---
## minder repo reconcile

Reconcile (Sync) a repository with Minder.

### Synopsis

The reconcile command is used to trigger a reconciliation (sync) of a repository against
profiles and rules in a project.

```
minder repo reconcile [flags]
```

### Options

```
  -h, --help          help for reconcile
  -i, --id string     ID of the repository
  -n, --name string   Name of the repository (owner/repo)
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

