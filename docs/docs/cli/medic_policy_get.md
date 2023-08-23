## medic policy get

Get details for a policy within a mediator control plane

### Synopsis

The medic policy get subcommand lets you retrieve details for a policy within a
mediator control plane.

```
medic policy get [flags]
```

### Options

```
  -h, --help              help for get
  -i, --id int32          ID for the policy to query
  -o, --output string     Output format (json or yaml)
  -p, --provider string   Provider for the policy (default "github")
  -s, --status            Only return the status of the policy for all the associated repos
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

