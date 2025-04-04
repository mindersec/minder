---
title: minder auth logout
---
## minder auth logout

Logout from minder control plane.

### Synopsis

Logout from minder control plane. Credentials will be removed from $XDG_CONFIG_HOME/minder/

```
minder auth logout [flags]
```

### Options

```
  -h, --help   help for logout
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.stacklok.com")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder auth](minder_auth.md)	 - Authorize and manage accounts within a minder control plane

