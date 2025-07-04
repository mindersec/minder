---
title: minder profile status get
---
## minder profile status get

Get profile status

### Synopsis

The profile status get subcommand lets you get profile status within Minder.

```
minder profile status get [flags]
```

### Options

```
      --emoji                Use emojis in the output (default true)
  -e, --entity string        Entity ID to get profile status for
  -t, --entity-type string   the entity type to get profile status for (one of artifact, build, build_environment, pipeline_run, release, repository, task_run)
  -h, --help                 help for get
  -i, --id string            ID to get profile status for
  -n, --name string          Profile name to get profile status for
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.custcodian.dev")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.custcodian.dev")
  -o, --output string            Output format (one of json,yaml,table) (default "table")
  -j, --project string           ID of the project
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder profile status](minder_profile_status.md)	 - Manage profile status

