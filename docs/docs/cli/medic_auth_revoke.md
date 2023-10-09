## medic auth revoke

Revoke access tokens

### Synopsis

It can revoke access tokens for one user or for all.

```
medic auth revoke [flags]
```

### Options

```
  -a, --all             Revoke all tokens
  -h, --help            help for revoke
  -u, --user-id int32   User ID to revoke tokens
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

* [medic auth](medic_auth.md)	 - Authorize and manage accounts within a mediator control plane

