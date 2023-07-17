## medic policy create

Create a policy within a mediator control plane

### Synopsis

The medic policy create subcommand lets you create new policies for a group
within a mediator control plane.

```
medic policy create [flags]
```

### Options

```
  -d, --default           Use default and recommended schema for the policy.
  -f, --file string       Path to the YAML defining the policy (or - for stdin)
  -g, --group-id int32    ID of the group to where the policy belongs
  -h, --help              help for create
  -n, --provider string   Provider (github)
  -t, --type string       Type of policy - must be one valid policy type.
                          	Please check valid policy types with: medic policy_types list command
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "localhost")
      --grpc-port int      Server port (default 8090)
```

### SEE ALSO

* [medic policy](medic_policy.md)	 - Manage policies within a mediator control plane

