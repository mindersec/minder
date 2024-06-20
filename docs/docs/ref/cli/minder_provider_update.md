---
title: minder provider update
---
## minder provider update

Updates a provider's configuration

### Synopsis

The minder provider update command allows a user to update a provider's
configuration after enrollement.

```
minder provider update [flags]
```

### Options

```
  -h, --help                      help for update
  -n, --name string               Name of the provider.
  -s, --set-attribute strings     List of attributes to set in the config in <name>=<value> format
  -u, --unset-attribute strings   List of attributes to unset in the config in <name> format
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

* [minder provider](minder_provider.md)	 - Manage providers within a minder control plane

