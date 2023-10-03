## medic role list

List roles within a mediator control plane

### Synopsis

The medic role list subcommand lets you list roles within a
mediator control plane for an specific project.

```
medic role list [flags]
```

### Options

```
  -h, --help                help for list
  -l, --limit int32         Limit the number of results returned (default -1)
  -f, --offset int32        Offset the results returned
  -i, --org-id string       org id to list roles for
  -o, --output string       Output format (json or yaml)
  -g, --project-id string   project id to list roles for
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic role](medic_role.md)	 - Manage roles within a mediator control plane

