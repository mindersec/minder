---
title: minder profile create
---
## minder profile create

Create a profile within a minder control plane

### Synopsis

The minder profile create subcommand lets you create new profiles for a project
within a minder control plane.

```
minder profile create [flags]
```

### Options

```
  -f, --file string      Path to the YAML defining the profile (or - for stdin)
  -h, --help             help for create
  -p, --project string   Project to create the profile in
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

* [minder profile](minder_profile.md)	 - Manage profiles within a minder control plane

