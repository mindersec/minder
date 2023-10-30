## medic rule_type delete

Delete a rule type within a mediator control plane

### Synopsis

The medic rule type delete subcommand lets you delete profiles within a
mediator control plane.

```
medic rule_type delete [flags]
```

### Options

```
  -h, --help        help for delete
  -i, --id string   ID of rule type to delete
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

* [medic rule_type](medic_rule_type.md)	 - Manage rule types within a mediator control plane

