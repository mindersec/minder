## medic auth logout

Logout from mediator control plane.

### Synopsis

Logout from mediator control plane. Credentials will be removed from $XDG_CONFIG_HOME/mediator/credentials.json

```
medic auth logout [flags]
```

### Options

```
  -h, --help   help for logout
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic auth](medic_auth.md)	 - Authorize and manage accounts within a mediator control plane

