---
title: minder auth login
---
## minder auth login

Login to a minder control plane.

### Synopsis

Login to a minder control plane. Upon successful login, credentials
will be saved to $XDG_CONFIG_HOME/minder/credentials.json

```
minder auth login [flags]
```

### Options

```
  -h, --help   help for login
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

* [minder auth](minder_auth.md)	 - Authorize and manage accounts within a minder control plane

