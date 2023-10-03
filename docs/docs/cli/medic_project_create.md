## medic project create

Create a project within a mediator control plane

### Synopsis

The medic project create subcommand lets you create new projects within
a mediator control plane.

```
medic project create [flags]
```

### Options

```
  -d, --description string   Description of the project
  -h, --help                 help for create
  -i, --is_protected         Is the project protected
  -n, --name string          Name of the project
      --org-id string        Organization ID
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic project](medic_project.md)	 - Manage projects within a mediator control plane

