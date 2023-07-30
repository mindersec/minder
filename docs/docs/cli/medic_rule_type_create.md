## medic rule_type create

Create a rule type within a mediator control plane

### Synopsis

The medic rule type create subcommand lets you create new policies for a group
within a mediator control plane.

```
medic rule_type create [flags]
```

### Options

```
  -f, --file string   Path to the YAML defining the rule type (or - for stdin)
  -h, --help          help for create
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "localhost")
      --grpc-port int      Server port (default 8090)
```

### SEE ALSO

* [medic rule_type](medic_rule_type.md)	 - Manage rule types within a mediator control plane

