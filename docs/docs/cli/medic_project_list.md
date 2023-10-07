## medic project list

Get list of projects within a mediator control plane

### Synopsis

The medic project list subcommand lets you list projects within
a mediator control plane.

```
medic project list [flags]
```

### Options

```
  -h, --help            help for list
  -l, --limit int32     Limit the number of results returned (default -1)
  -f, --offset int32    Offset the results returned
  -i, --org-id string   Organisation ID to list projects for
  -o, --output string   Output format
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

