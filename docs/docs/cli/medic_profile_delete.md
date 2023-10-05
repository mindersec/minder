## medic profile delete

delete a profile within a mediator controlplane

### Synopsis

The medic profile delete subcommand lets you delete profiles within a
mediator control plane.

```
medic profile delete [flags]
```

### Options

```
  -h, --help              help for delete
  -i, --id string         id of profile to delete
  -p, --provider string   Provider for the profile (default "github")
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic profile](medic_profile.md)	 - Manage profiles within a mediator control plane

