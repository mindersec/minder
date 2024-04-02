---
title: minder profile create
---
## minder profile create

Create a profile

### Synopsis

The profile create subcommand lets you create new profiles for a project within Minder.

```
minder profile create [flags]
```

### Options

```
      --enable-alerts         Explicitly enable alerts for this profile. Overrides the YAML file.
      --enable-remediations   Explicitly enable remediations for this profile. Overrides the YAML file.
  -f, --file string           Path to the YAML defining the profile (or - for stdin)
  -h, --help                  help for create
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

