## medic repo register

Register a repo with the mediator control plane

### Synopsis

Repo register is used to register a repo with the mediator control plane

```
medic repo register [flags]
```

### Options

```
  -g, --group-id int32    ID of the group for repo registration
  -h, --help              help for register
  -l, --limit int32       Number of repos to display per page (default 20)
  -o, --offset int32      Offset of the repos to display
  -n, --provider string   Name for the provider to enroll
      --repo string       List of key-value pairs
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic repo](medic_repo.md)	 - Manage repositories within a mediator control plane

