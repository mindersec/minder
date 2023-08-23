## medic rule_type get

Get details for a rule type within a mediator control plane

### Synopsis

The medic rule_type get subcommand lets you retrieve details for a rule type within a
mediator control plane.

```
medic rule_type get [flags]
```

### Options

```
  -h, --help              help for get
  -i, --id int32          ID for the policy to query
  -o, --output string     Output format (json or yaml) (default "yaml")
  -p, --provider string   Provider for the policy (default "github")
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "localhost")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 8090)
```

### SEE ALSO

* [medic rule_type](medic_rule_type.md)	 - Manage rule types within a mediator control plane

