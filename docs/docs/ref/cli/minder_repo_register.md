---
title: minder repo register
---
## minder repo register

Register a repository

### Synopsis

The repo register subcommand is used to register a repo within Minder.

```
minder repo register [flags]
```

### Options

```
  -h, --help          help for register
  -n, --name string   List of repository names to register, i.e owner/repo,owner/repo
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

