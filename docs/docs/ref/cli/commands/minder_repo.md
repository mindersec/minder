---
title: minder repo
---
## minder repo

Manage repositories within a minder control plane

### Synopsis

The minder repo commands allow the management of repositories within a 
minder control plane.

```
minder repo [flags]
```

### Options

```
  -h, --help   help for repo
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

* [minder](minder.md)	 - Minder controls the hosted minder service
* [minder repo delete](minder_repo_delete.md)	 - delete repository
* [minder repo get](minder_repo_get.md)	 - Get repository in the minder control plane
* [minder repo list](minder_repo_list.md)	 - List repositories in the minder control plane
* [minder repo register](minder_repo_register.md)	 - Register a repo with the minder control plane

