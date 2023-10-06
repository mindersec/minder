## medic project delete

Delete a project within a mediator control plane

### Synopsis

The medic project delete subcommand lets you delete projects within a
mediator control plane.

```
medic project delete [flags]
```

### Options

```
  -f, --force               Force deletion of project, even if it's protected or has associated roles (WARNING: removing a protected project may cause loosing mediator access)
  -h, --help                help for delete
  -g, --project-id string   ID of project to delete
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

