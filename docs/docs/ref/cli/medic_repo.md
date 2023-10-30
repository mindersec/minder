## medic repo

Manage repositories within a mediator control plane

### Synopsis

The medic repo commands allow the management of repositories within a 
mediator control plane.

```
medic repo [flags]
```

### Options

```
  -h, --help   help for repo
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "staging.stacklok.dev")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "mediator-cli")
      --identity-realm string    Identity server realm (default "stacklok")
      --identity-url string      Identity server issuer URL (default "https://auth.staging.stacklok.dev")
```

### SEE ALSO

* [medic](medic.md)	 - Medic controls mediator via the control plane
* [medic repo get](medic_repo_get.md)	 - Get repository in the mediator control plane
* [medic repo list](medic_repo_list.md)	 - List repositories in the mediator control plane
* [medic repo register](medic_repo_register.md)	 - Register a repo with the mediator control plane

