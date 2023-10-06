## medic auth login

Login to a mediator control plane.

### Synopsis

Login to a mediator control plane. Upon successful login, credentials
will be saved to $XDG_CONFIG_HOME/mediator/credentials.json

```
medic auth login [flags]
```

### Options

```
  -h, --help   help for login
```

### Options inherited from parent commands

```
      --config string      Config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic auth](medic_auth.md)	 - Authorize and manage accounts within a mediator control plane

