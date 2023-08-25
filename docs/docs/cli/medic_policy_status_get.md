## medic policy_status get

Get policy status within a mediator control plane

### Synopsis

The medic policy_status get subcommand lets you get policy status within a
mediator control plane for an specific provider/group or policy id, entity type and entity id.

```
medic policy_status get [flags]
```

### Options

```
  -e, --entity int32         entity id to get policy status for
  -t, --entity-type string   the entity type to get policy status for (one of repository,build_environment,artifact)
  -g, --group string         group id to get policy status for
  -h, --help                 help for get
  -o, --output string        Output format (json or yaml) (default "yaml")
  -i, --policy int32         policy id to get policy status for
  -p, --provider string      Provider to get policy status for (default "github")
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

