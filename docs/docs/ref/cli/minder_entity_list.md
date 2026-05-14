---
title: minder entity list
---
## minder entity list

List entities

### Synopsis

The entity list subcommand is used to list entity instances within Minder.

```
minder entity list [flags]
```

### Options

```
      --emoji              Use emojis in the output (default true)
  -h, --help               help for list
  -o, --output string      Output format (one of json,yaml,table) (default "table")
      --property strings   Properties to include in the output table
  -t, --type string        Type of entity to list (e.g. repository, artifact, pull_request)
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

