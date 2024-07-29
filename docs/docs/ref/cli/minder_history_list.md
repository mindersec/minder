---
title: minder history list
---
## minder history list

List history

### Synopsis

The history list subcommand lets you list history within Minder.

```
minder history list [flags]
```

### Options

```
      --alert-status strings         Filter evaluation history list by alert status - one of off, on, error, skipped, not_available
  -c, --cursor string                Fetch previous or next page from the list
      --entity-name strings          Filter evaluation history list by entity name
      --entity-type strings          Filter evaluation history list by entity type - one of repository, artifact, pull_request
      --eval-status strings          Filter evaluation history list by evaluation status - one of pending, failure, error, success, skipped
      --from string                  Filter evaluation history list by time
  -h, --help                         help for list
      --profile-name strings         Filter evaluation history list by profile name
      --remediation-status strings   Filter evaluation history list by remediation status - one of failure, failure, error, success, skipped, not_available
  -s, --size uint                    Change the number of items fetched (default 25)
      --to string                    Filter evaluation history list by time
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.stacklok.com")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")
  -o, --output string            Output format (one of json,yaml,table) (default "table")
  -j, --project string           ID of the project
```

### SEE ALSO

* [minder history](minder_history.md)	 - View evaluation history

