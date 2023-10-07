## medic rule_type list

List rule types within a mediator control plane

### Synopsis

The medic rule_type list subcommand lets you list rule type within a
mediator control plane for an specific project.

```
medic rule_type list [flags]
```

### Options

```
  -h, --help              help for list
  -o, --output string     Output format (json, yaml or table) (default "table")
  -p, --provider string   Provider to list rule types for
```

### Options inherited from parent commands

```
      --config string      Config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic rule_type](medic_rule_type.md)	 - Manage rule types within a mediator control plane

