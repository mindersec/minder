---
title: minder auth offline-token get
---
## minder auth offline-token get

Retrieve an offline token

### Synopsis

The minder auth offline-token get command project lets you retrieve an offline token
for the minder control plane.

Offline tokens are used to authenticate to the minder control plane without
requiring the user's presence. This is useful for long-running processes
that need to authenticate to the control plane.

```
minder auth offline-token get [flags]
```

### Options

```
  -f, --file string   The file to write the offline token to (default "offline.token")
  -h, --help          help for get
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

