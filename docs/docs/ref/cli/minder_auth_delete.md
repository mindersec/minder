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
  -h, --help                    help for delete
      --yes-delete-my-account   Bypass yes/no prompt when deleting the account
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.custcodian.dev")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.custcodian.dev")
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder auth](minder_auth.md)	 - Authorize and manage accounts within a minder control plane

