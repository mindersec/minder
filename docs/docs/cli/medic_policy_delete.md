## medic policy delete

delete a policy within a mediator controlplane

### Synopsis

The medic policy delete subcommand lets you delete policies within a
mediator control plane.

```
medic policy delete [flags]
```

### Options

```
  -h, --help              help for delete
  -i, --id int32          id of policy to delete
  -p, --provider string   Provider for the policy (default "github")
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

