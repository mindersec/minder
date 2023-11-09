---
title: minder profile
---
## minder profile

Manage profiles within a minder control plane

### Synopsis

The minder profile subcommands allows the management of profiles within
a minder controlplane.

```
minder profile [flags]
```

### Options

```
  -h, --help   help for profile
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
* [minder profile create](minder_profile_create.md)	 - Create a profile within a minder control plane
* [minder profile delete](minder_profile_delete.md)	 - Delete a profile within a minder control plane
* [minder profile get](minder_profile_get.md)	 - Get details for a profile within a minder control plane
* [minder profile list](minder_profile_list.md)	 - List profiles within a minder control plane

