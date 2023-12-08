---
title: minder profile status
---
## minder profile status

Manage profile status within a minder control plane

### Synopsis

The minder profile status subcommand allows the management of profile status within
a minder control plane.

```
minder profile status [flags]
```

### Options

```
  -h, --help   help for status
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
* [minder profile status get](minder_profile_status_get.md)	 - Get profile status within a minder control plane
* [minder profile status list](minder_profile_status_list.md)	 - List profile status within a minder control plane

