## medic policy list

List policies within a mediator control plane

### Synopsis

The medic policy list subcommand lets you list policies within a
mediator control plane for an specific project.

```
medic policy list [flags]
```

### Options

```
  -h, --help              help for list
  -o, --output string     Output format (json, yaml or table) (default "table")
  -p, --provider string   Provider to list policies for
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic policy](medic_policy.md)	 - Manage policies within a mediator control plane

