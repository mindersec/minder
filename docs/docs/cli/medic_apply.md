## medic apply

Appy a configuration to a mediator control plane

### Synopsis

The medic apply command applies a configuration to a mediator control plane.

```
medic apply (-f FILENAME) [flags]
```

### Options

```
  -f, --file string   Path to the configuration file to apply or - for stdin
  -h, --help          help for apply
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic](medic.md)	 - medic controls mediator via the control plane

