## medic org delete

Delete an organization within a mediator control plane

### Synopsis

The medic org delete subcommand lets you delete organizations within a
mediator control plane.

```
medic org delete [flags]
```

### Options

```
  -f, --force           Force deletion of organization, even if it has associated projects
  -h, --help            help for delete
  -o, --org-id string   ID of organization to delete
```

### Options inherited from parent commands

```
      --config string      Config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic org](medic_org.md)	 - Manage organizations within a mediator control plane

