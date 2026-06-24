---
title: minder entity register
---
## minder entity register

Register an entity

### Synopsis

The entity register subcommand is used to register a new entity instance within Minder.

Identifying properties are specified as key=value pairs using the --property flag.
For example, for a GitHub repository:
  --property github/repo_owner=myorg --property github/repo_name=myrepo

```
minder entity register [flags]
```

### Examples

```

  # Register a GitHub repository
    minder entity register --type repository --property github/repo_owner=myorg --property github/repo_name=myrepo

```

### Options

```
  -h, --help                   help for register
  -o, --output string          Output format (one of json,yaml,table) (default "table")
  -P, --property stringArray   Identifying property in key=value format (may be repeated)
  -t, --type string            Type of entity to register (e.g. repository, artifact, pull_request)
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.custcodian.dev")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.custcodian.dev")
  -j, --project string           ID of the project
  -p, --provider string          Name of the provider, i.e. github
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder entity](minder_entity.md)	 - Manage entities within a Minder project

