## medic auth

Authorize and manage accounts within a mediator control plane

### Synopsis

The medic auth command project lets you create accounts and grant or revoke
authorization to existing accounts within a mediator control plane.

```
medic auth [flags]
```

### Options

```
  -h, --help   help for auth
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "staging.stacklok.dev")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "mediator-cli")
      --identity-realm string    Identity server realm (default "stacklok")
      --identity-url string      Identity server issuer URL (default "http://localhost:8081")
```

### SEE ALSO

* [medic](medic.md)	 - Medic controls mediator via the control plane
* [medic auth login](medic_auth_login.md)	 - Login to a mediator control plane.
* [medic auth logout](medic_auth_logout.md)	 - Logout from mediator control plane.
* [medic auth revoke_provider](medic_auth_revoke_provider.md)	 - Revoke access tokens for provider

