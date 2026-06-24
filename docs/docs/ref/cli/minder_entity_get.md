---
title: minder entity get
---
## minder entity get

Get entity details

### Synopsis

The entity get subcommand is used to get details for an entity instance within Minder.

```
minder entity get [flags]
```

### Options

```
      --emoji           Use emojis in the output (default true)
  -h, --help            help for get
  -i, --id string       ID of the entity to get
  -n, --name string     Name of the entity to get
  -o, --output string   Output format (one of table,json,yaml) (default "table")
  -t, --type string     Type of entity (e.g. repository, artifact, pull_request); required with --name
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

