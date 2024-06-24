---
title: minder auth
---
## minder auth

Authorize and manage accounts within a minder control plane

### Synopsis

The minder auth command project lets you create accounts and grant or revoke
authorization to existing accounts within a minder control plane.

```
minder auth [flags]
```

### Options

```
  -h, --help   help for auth
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

* [minder](minder.md)	 - Minder controls the hosted minder service
* [minder auth delete](minder_auth_delete.md)	 - Permanently delete account
* [minder auth invite](minder_auth_invite.md)	 - Manage user invitations
* [minder auth login](minder_auth_login.md)	 - Login to Minder
* [minder auth logout](minder_auth_logout.md)	 - Logout from minder control plane.
* [minder auth offline-token](minder_auth_offline-token.md)	 - Manage offline tokens
* [minder auth token](minder_auth_token.md)	 - Print your token for Minder
* [minder auth whoami](minder_auth_whoami.md)	 - whoami for current user

