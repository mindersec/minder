## medic role create

Create a role within a mediator control plane

### Synopsis

The medic role create subcommand lets you create new roles for a project
within a mediator control plane.

```
medic role create [flags]
```

### Options

```
  -h, --help                help for create
  -a, --is_admin            Is it an admin role
  -i, --is_protected        Is the role protected
  -n, --name string         Name of the role
  -o, --org-id string       ID of the organization which owns the role
  -g, --project-id string   ID of the project which owns the role
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

