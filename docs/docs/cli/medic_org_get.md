## medic org get

Get details for an organization within a mediator control plane

### Synopsis

The medic org get subcommand lets you retrieve details for an organization within a
mediator control plane.

```
medic org get [flags]
```

### Options

```
  -h, --help            help for get
  -i, --id string       ID for the organization to query
  -n, --name string     Name for the organization to query
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

* [medic org](medic_org.md)	 - Manage organizations within a mediator control plane

