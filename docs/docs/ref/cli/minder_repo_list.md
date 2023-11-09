---
title: minder repo list
---
## minder repo list

List repositories in the minder control plane

### Synopsis

Repo list is used to register a repo with the minder control plane

```
minder repo list [flags]
```

### Options

```
  -h, --help                help for list
  -f, --output string       Output format (json or yaml)
  -g, --project-id string   ID of the project for repo registration
  -p, --provider string     Name for the provider to enroll
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

