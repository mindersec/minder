## medic role list

List roles within a mediator control plane

### Synopsis

The medic role list subcommand lets you list roles within a
mediator control plane for an specific group.

```
medic role list [flags]
```

### Options

```
  -g, --group-id int32   group id to list roles for
  -h, --help             help for list
  -l, --limit int32      Limit the number of results returned (default -1)
  -f, --offset int32     Offset the results returned
  -i, --org-id int32     org id to list roles for
  -o, --output string    Output format (json or yaml)
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "localhost")
      --grpc-port int      Server port (default 8090)
```

### SEE ALSO

* [medic role](medic_role.md)	 - Manage roles within a mediator control plane

