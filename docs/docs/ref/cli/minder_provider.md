---
title: minder provider
---
## minder provider

Manage providers within a minder control plane

### Synopsis

The minder provider commands manage providers within a minder control plane.

```
minder provider [flags]
```

### Options

```
  -h, --help             help for provider
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
  -v, --verbose                  Output additional messages to STDERR
```

### SEE ALSO

* [minder](minder.md)	 - Minder controls the hosted minder service
* [minder provider delete](minder_provider_delete.md)	 - Delete a given provider available in a specific project
* [minder provider enroll](minder_provider_enroll.md)	 - Enroll a provider within the minder control plane
* [minder provider get](minder_provider_get.md)	 - Get a given provider available in a specific project
* [minder provider list](minder_provider_list.md)	 - List the providers available in a specific project
* [minder provider update](minder_provider_update.md)	 - Updates a provider's configuration

