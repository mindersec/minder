## medic project get

Get details for an project within a mediator control plane

### Synopsis

The medic project get subcommand lets you retrieve details for a project within a
mediator control plane.

```
medic project get [flags]
```

### Options

```
  -h, --help            help for get
  -i, --id string       ID for the project to query
  -n, --name string     Name for the project to query
  -o, --output string   Output format (json or yaml)
```

### Options inherited from parent commands

```
      --config string      Config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic project](medic_project.md)	 - Manage projects within a mediator control plane

