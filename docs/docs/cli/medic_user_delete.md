## medic user delete

delete a user within a mediator controlplane

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
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "localhost")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 8090)
```

### SEE ALSO

* [medic user](medic_user.md)	 - Manage users within a mediator control plane

