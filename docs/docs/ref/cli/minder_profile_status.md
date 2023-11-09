---
title: minder profile status
---
## minder profile_status

Manage profile status within a minder control plane

### Synopsis

The minder profile_status subcommands allows the management of profile status within
a minder control plane.

```
minder profile_status [flags]
```

### Options

```
  -h, --help   help for profile_status
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
* [minder profile_status get](minder_profile_status_get.md)	 - Get profile status within a minder control plane
* [minder profile_status list](minder_profile_status_list.md)	 - List profile status within a minder control plane

