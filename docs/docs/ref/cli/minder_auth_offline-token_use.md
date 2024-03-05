---
title: minder auth offline-token use
---
## minder auth offline-token use

Use an offline token

### Synopsis

The minder auth offline-token use command project lets you install and use an offline token
for the minder control plane.

Offline tokens are used to authenticate to the minder control plane without
requiring the user's presence. This is useful for long-running processes
that need to authenticate to the control plane.

```
minder auth offline-token use [flags]
```

### Options

```
  -f, --file string    The file that contains the offline token (default "offline.token")
  -h, --help           help for use
  -t, --token string   The environment variable to use for the offline token. Also settable through the MINDER_OFFLINE_TOKEN environment variable.
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

* [minder auth offline-token](minder_auth_offline-token.md)	 - Manage offline tokens

