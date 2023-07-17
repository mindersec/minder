## medic org list

List organizations within a mediator control plane

### Synopsis

The medic org list subcommand lets you list organizations within a
mediator control plane.

```
medic org list [flags]
```

### Options

```
  -h, --help            help for list
  -l, --limit int32     Limit the number of results returned (default -1)
  -f, --offset int32    Offset the results returned
  -o, --output string   Output format (json or yaml)
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "localhost")
      --grpc-port int      Server port (default 8090)
```

### SEE ALSO

* [medic org](medic_org.md)	 - Manage organizations within a mediator control plane

