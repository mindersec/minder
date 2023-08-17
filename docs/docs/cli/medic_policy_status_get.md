## medic policy_status get

Get policy status within a mediator control plane

### Synopsis

The medic policy_status get subcommand lets you get policy status within a
mediator control plane for an specific provider/group or policy id and repo-id.

```
medic policy_status get [flags]
```

### Options

```
  -g, --group string      group id to get policy status for
  -h, --help              help for get
  -o, --output string     Output format (json or yaml) (default "yaml")
  -i, --policy int32      policy id to get policy status for
  -p, --provider string   Provider to get policy status for (default "github")
  -r, --repo int32        repo id to get policy status for
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "localhost")
      --grpc-port int      Server port (default 8090)
```

### SEE ALSO

* [medic policy_status](medic_policy_status.md)	 - Manage policy status within a mediator control plane

