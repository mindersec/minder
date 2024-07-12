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
      --alert-status string         Filter evaluation history list by alert status - one of off, on, error, skipped, not_available
      --entity-name string          Filter evaluation history list by entity name
      --entity-type string          Filter evaluation history list by entity type - one of repository, artifact, pull_request
      --eval-status string          Filter evaluation history list by evaluation status - one of pending, failure, error, success, skipped
  -h, --help                        help for list
      --profile-name string         Filter evaluation history list by profile name
      --remediation-status string   Filter evaluation history list by remediation status - one of failure, failure, error, success, skipped, not_available
      --rule-name string            Filter evaluation history list by rule name
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

