## medic group delete

delete a group within a mediator controlplane

### Synopsis

The medic group delete subcommand lets you delete groups within a
mediator control plane.

```
medic group delete [flags]
```

### Options

```
  -f, --force            Force deletion of group, even if it's protected or has associated roles (WARNING: removing a protected group may cause loosing mediator access)
  -g, --group-id int32   id of group to delete
  -h, --help             help for delete
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

