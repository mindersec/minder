---
title: minder auth login
---
## minder auth login

Login to Minder

### Synopsis

The login command allows for logging in to Minder. Upon successful login, credentials will be saved to
$XDG_CONFIG_HOME/minder/credentials.json

```
minder auth login [flags]
```

### Options

```
  -h, --help           help for login
      --skip-browser   Skip opening the browser for OAuth flow
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

