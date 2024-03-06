---
title: minder auth offline-token
---
## minder auth offline-token

Manage offline tokens

### Synopsis

The minder auth offline-token command project lets you manage offline tokens
for the minder control plane.

Offline tokens are used to authenticate to the minder control plane without
requiring the user's presence. This is useful for long-running processes
that need to authenticate to the control plane.

```
minder auth offline-token [flags]
```

### Options

```
  -h, --help   help for offline-token
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

* [minder auth](minder_auth.md)	 - Authorize and manage accounts within a minder control plane
* [minder auth offline-token get](minder_auth_offline-token_get.md)	 - Retrieve an offline token
* [minder auth offline-token revoke](minder_auth_offline-token_revoke.md)	 - Revoke an offline token
* [minder auth offline-token use](minder_auth_offline-token_use.md)	 - Use an offline token

