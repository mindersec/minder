## medic keys generate

Generate keys within a mediator control plane

### Synopsis

The medic keys generate  subcommand lets you create keys within a
mediator control plane for an specific group.

```
medic keys generate [flags]
```

### Options

```
  -g, --group-id int32      group id to list roles for
  -h, --help                help for generate
  -o, --output string       Output public key to file
  -p, --passphrase string   Passphrase to use for key generation
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic keys](medic_keys.md)	 - Manage keys within a mediator control plane

