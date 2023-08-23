## medic artifact list

List artifacts from a provider

### Synopsis

Artifact list will list artifacts from a provider

```
medic artifact list [flags]
```

### Options

```
  -g, --group-id int32    ID of the group for repo registration
  -h, --help              help for list
  -f, --output string     Output format (json or yaml)
  -n, --provider string   Name for the provider to enroll
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic artifact](medic_artifact.md)	 - Manage repositories within a mediator control plane

