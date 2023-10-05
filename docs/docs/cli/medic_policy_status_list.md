## medic policy_status list

List policy status within a mediator control plane

### Synopsis

The medic policy_status list subcommand lets you list policy status within a
mediator control plane for an specific provider/project or policy id.

```
medic policy_status list [flags]
```

### Options

```
  -d, --detailed          List all policy violations
  -h, --help              help for list
  -o, --output string     Output format (json, yaml or table) (default "table")
  -i, --policy string     policy name to list policy status for
  -g, --project string    project id to list policy status for
  -p, --provider string   Provider to list policy status for (default "github")
  -r, --rule string       Filter policy status list by rule
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic policy_status](medic_policy_status.md)	 - Manage policy status within a mediator control plane

