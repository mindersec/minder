---
title: minder profile
---
## minder profile

Manage profiles

### Synopsis

The profile subcommands allows the management of profiles within Minder.

```
minder profile [flags]
```

### Options

```
  -h, --help             help for profile
  -j, --project string   ID of the project
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.stacklok.com")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")
```

### SEE ALSO

* [minder](minder.md)	 - Minder controls the hosted minder service
* [minder profile apply](minder_profile_apply.md)	 - Create or update a profile
* [minder profile create](minder_profile_create.md)	 - Create a profile
* [minder profile delete](minder_profile_delete.md)	 - Delete a profile
* [minder profile get](minder_profile_get.md)	 - Get details for a profile
* [minder profile list](minder_profile_list.md)	 - List profiles
* [minder profile status](minder_profile_status.md)	 - Manage profile status

