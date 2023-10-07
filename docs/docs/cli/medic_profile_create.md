## medic profile create

Create a profile within a mediator control plane

### Synopsis

The medic profile create subcommand lets you create new profiles for a project
within a mediator control plane.

```
medic profile create [flags]
```

### Options

```
  -f, --file string      Path to the YAML defining the profile (or - for stdin)
  -h, --help             help for create
  -p, --project string   Project to create the profile in
```

### Options inherited from parent commands

```
      --config string      Config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic profile](medic_profile.md)	 - Manage profiles within a mediator control plane

