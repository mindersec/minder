---
title: minder auth delete
---
## minder auth delete

Permanently delete account

### Synopsis

Permanently delete account. All associated user data will be permanently removed.

```
minder auth delete [flags]
```

### Options

```
  -h, --help   help for delete
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "staging.stacklok.dev")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-realm string    Identity server realm (default "stacklok")
      --identity-url string      Identity server issuer URL (default "https://auth.staging.stacklok.dev")
```

### SEE ALSO

* [minder auth](minder_auth.md)	 - Authorize and manage accounts within a minder control plane

