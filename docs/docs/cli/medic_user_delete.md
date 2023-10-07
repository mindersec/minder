## medic user delete

Delete an user within a mediator control plane

### Synopsis

The medic user delete subcommand lets you delete users within a
mediator control plane.

```
medic user delete [flags]
```

### Options

```
  -f, --force           Force deletion of user, even if it's protected (WARNING: removing a protected user may cause loss of mediator access and data)
  -h, --help            help for delete
  -u, --user-id int32   id of user to delete
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

