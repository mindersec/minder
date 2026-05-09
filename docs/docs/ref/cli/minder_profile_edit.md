---
title: minder profile edit
---
## minder profile edit

Edit an existing profile

### Synopsis

The profile edit subcommand lets you fetch an existing profile, edit it in your $EDITOR, and apply the updates.

```
minder profile edit [flags]
```

### Options

```
  -h, --help          help for edit
  -i, --id string     ID of the profile to edit
  -n, --name string   Name of the profile to edit
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
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder profile](minder_profile.md)	 - Manage profiles

