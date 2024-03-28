---
title: minder ruletype create
---
## minder ruletype create

Create a rule type

### Synopsis

The ruletype create subcommand lets you create new rule types for a project within Minder.

```
minder ruletype create [flags]
```

### Options

```
  -f, --file stringArray   Path to the YAML defining the rule type (or - for stdin). Can be specified multiple times. Can be a directory.
  -h, --help               help for create
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.stacklok.com")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")
  -j, --project string           ID of the project
```

### SEE ALSO

* [minder ruletype](minder_ruletype.md)	 - Manage rule types

