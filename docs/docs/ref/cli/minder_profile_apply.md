---
title: minder profile apply
---
## minder profile apply

Create or update a profile within a minder control plane

### Synopsis

The minder profile apply subcommand lets you create or update new profiles for a project
within a minder control plane.

```
minder profile apply [flags]
```

### Options

```
  -f, --file string      Path to the YAML defining the profile (or - for stdin)
  -h, --help             help for apply
  -p, --project string   Project to create the profile in
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

* [minder profile](minder_profile.md)	 - Manage profiles within a minder control plane

