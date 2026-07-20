---
title: minder entity
---
## minder entity

Manage entities within a Minder project

### Synopsis

Manage entities within a Minder project.

This command allows you to list, get, register, and delete entity instances
connected to Minder for security analysis and policy enforcement.

```
minder entity [flags]
```

### Examples

```

  # List entities
    minder entity list --type repository

  # Get an entity by ID
    minder entity get --id <entity-id>

  # Register an entity
    minder entity register --type repository --property github/repo_owner=owner --property github/repo_name=name

  # Delete an entity
    minder entity delete --id <entity-id>

```

### Options

```
  -h, --help              help for entity
  -j, --project string    ID of the project
  -p, --provider string   Name of the provider, i.e. github
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.custcodian.dev")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.custcodian.dev")
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder](minder.md)	 - Minder controls the hosted minder service
* [minder entity delete](minder_entity_delete.md)	 - Delete an entity
* [minder entity get](minder_entity_get.md)	 - Get entity details
* [minder entity list](minder_entity_list.md)	 - List entities
* [minder entity register](minder_entity_register.md)	 - Register an entity

