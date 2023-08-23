## medic org delete

delete a organization within a mediator controlplane

### Synopsis

The medic org delete subcommand lets you delete organizations within a
mediator control plane.

```
medic org delete [flags]
```

### Options

```
  -f, --force          Force deletion of organization, even if it has associated groups
  -h, --help           help for delete
  -o, --org-id int32   id of organization to delete
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "localhost")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 8090)
```

### SEE ALSO

* [medic org](medic_org.md)	 - Manage organizations within a mediator control plane

