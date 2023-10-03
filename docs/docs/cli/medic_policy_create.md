## medic policy create

Create a policy within a mediator control plane

### Synopsis

The medic policy create subcommand lets you create new policies for a project
within a mediator control plane.

```
medic policy create [flags]
```

### Options

```
  -f, --file string   Path to the YAML defining the policy (or - for stdin)
  -h, --help          help for create
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic policy](medic_policy.md)	 - Manage policies within a mediator control plane

