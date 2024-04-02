---
title: minder profile delete
---
## minder profile delete

Delete a profile

### Synopsis

The profile delete subcommand lets you delete profiles within Minder.

```
minder profile delete [flags]
```

### Options

```
  -h, --help        help for delete
  -i, --id string   ID of profile to delete
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

* [minder profile](minder_profile.md)	 - Manage profiles

