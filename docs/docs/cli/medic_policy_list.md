## medic policy list

List policies within a mediator control plane

### Synopsis

The medic policy list subcommand lets you list policies within a
mediator control plane for an specific group.

```
medic policy list [flags]
```

### Options

```
  -h, --help              help for list
  -o, --output string     Output format (json or yaml) (default "yaml")
  -p, --provider string   Provider to list policies for
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "localhost")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 8090)
```

### SEE ALSO

* [medic policy](medic_policy.md)	 - Manage policies within a mediator control plane

