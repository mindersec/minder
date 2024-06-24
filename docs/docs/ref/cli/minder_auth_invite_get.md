---
title: minder auth invite get
---
## minder auth invite get

Get info about pending invitations

### Synopsis

Get shows additional information about a pending invitation

```
minder auth invite get [flags]
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format (one of json,yaml,table) (default "table")
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

* [minder auth invite](minder_auth_invite.md)	 - Manage user invitations

