## medic policy_status list

List policy status within a mediator control plane

### Synopsis

The medic policy_status list subcommand lets you list policy status within a
mediator control plane for an specific provider/group or policy id.

```
medic policy_status list [flags]
```

### Options

```
  -a, --all               List all policy violations
  -g, --group string      group id to list policy status for
  -h, --help              help for list
  -o, --output string     Output format (json or yaml) (default "yaml")
  -i, --policy int32      policy id to list policy status for
  -p, --provider string   Provider to list policy status for (default "github")
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "localhost")
      --grpc-port int      Server port (default 8090)
```

### SEE ALSO

* [medic policy_status](medic_policy_status.md)	 - Manage policy status within a mediator control plane

