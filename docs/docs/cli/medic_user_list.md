## medic user list

List users within a mediator control plane

### Synopsis

The medic user list subcommand lets you list users within a
mediator control plane for an specific role.

```
medic user list [flags]
```

### Options

```
  -h, --help                help for list
  -l, --limit int32         Limit the number of results returned (default -1)
  -f, --offset int32        Offset the results returned
  -i, --org-id string       org id to list users for
  -o, --output string       Output format (json or yaml)
  -g, --project-id string   project id to list users for
```

### Options inherited from parent commands

```
      --config string      Config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic user](medic_user.md)	 - Manage users within a mediator control plane

