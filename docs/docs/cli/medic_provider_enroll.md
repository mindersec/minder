## medic provider enroll

Enroll a provider within the mediator control plane

### Synopsis

The medic provider enroll command allows a user to enroll a provider
such as GitHub into the mediator control plane. Once enrolled, users can perform
actions such as adding repositories.

```
medic provider enroll [flags]
```

### Options

```
  -g, --group-id int32    ID of the group for enrolling the provider
  -h, --help              help for enroll
  -o, --owner string      Owner to filter on for provider resources
  -n, --provider string   Name for the provider to enroll
  -t, --token string      Personal Access Token (PAT) to use for enrollment
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "localhost")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 8090)
```

### SEE ALSO

* [medic provider](medic_provider.md)	 - Manage providers within a mediator control plane

