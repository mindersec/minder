---
title: minder auth invite
---
## minder auth invite

Manage user invitations

### Synopsis

The minder auth invite command lets you manage (accept/decline/list) your invitations.

```
minder auth invite [flags]
```

### Options

```
  -h, --help   help for invite
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
* [minder auth invite accept](minder_auth_invite_accept.md)	 - Accept a pending invitation
* [minder auth invite decline](minder_auth_invite_decline.md)	 - Declines a pending invitation
* [minder auth invite get](minder_auth_invite_get.md)	 - Get info about pending invitations
* [minder auth invite list](minder_auth_invite_list.md)	 - List pending invitations

