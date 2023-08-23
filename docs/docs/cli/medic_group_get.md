## medic group get

Get details for an group within a mediator control plane

### Synopsis

The medic group get subcommand lets you retrieve details for a group within a
mediator control plane.

```
medic group get [flags]
```

### Options

```
  -g, --group-id int32   Group ID
  -h, --help             help for get
  -i, --id int32         ID for the role to query
  -n, --name string      Name for the role to query
  -o, --output string    Output format (json or yaml)
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic group](medic_group.md)	 - Manage groups within a mediator control plane

