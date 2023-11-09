---
title: minder repo delete
---
## minder repo delete

delete repository

### Synopsis

Repo delete is used to delete a repository within the minder control plane

```
minder repo delete [flags]
```

### Options

```
  -h, --help              help for delete
  -n, --name string       Name of the repository (owner/name format)
  -p, --provider string   Name of the enrolled provider
  -r, --repo-id string    ID of the repo to delete
  -s, --status            Only return the status of the profiles associated to this repo
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.stacklok.com")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-realm string    Identity server realm (default "stacklok")
      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")
```

### SEE ALSO

* [minder repo](minder_repo.md)	 - Manage repositories within a minder control plane

