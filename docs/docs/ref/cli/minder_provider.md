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
  -h, --help   help for provider
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.stacklok.com")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-realm string    Identity server realm (default "stacklok")
      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")
```

### SEE ALSO

* [minder](minder.md)	 - Minder controls the hosted minder service
* [minder provider enroll](minder_provider_enroll.md)	 - Enroll a provider within the minder control plane

