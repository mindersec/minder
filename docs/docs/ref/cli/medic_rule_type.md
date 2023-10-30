## medic rule_type

Manage rule types within a mediator control plane

### Synopsis

The medic rule_type subcommands allows the management of rule types within
a mediator controlplane.

```
medic rule_type [flags]
```

### Options

```
  -h, --help   help for rule_type
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "staging.stacklok.dev")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "mediator-cli")
      --identity-realm string    Identity server realm (default "stacklok")
      --identity-url string      Identity server issuer URL (default "https://auth.staging.stacklok.dev")
```

### SEE ALSO

* [medic](medic.md)	 - Medic controls mediator via the control plane
* [medic rule_type create](medic_rule_type_create.md)	 - Create a rule type within a mediator control plane
* [medic rule_type delete](medic_rule_type_delete.md)	 - Delete a rule type within a mediator control plane
* [medic rule_type get](medic_rule_type_get.md)	 - Get details for a rule type within a mediator control plane
* [medic rule_type list](medic_rule_type_list.md)	 - List rule types within a mediator control plane

