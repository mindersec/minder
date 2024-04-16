---
title: minder auth token
---
## minder auth token

Print your token for Minder

### Synopsis

The token command allows for printing the token for Minder. This is useful
for using with automation scripts or other tools.

```
minder auth token [flags]
```

### Options

```
  -h, --help           help for token
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

