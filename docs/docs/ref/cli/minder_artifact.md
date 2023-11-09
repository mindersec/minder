---
title: minder artifact
---
## minder artifact

Manage artifacts within a minder control plane

### Synopsis

The minder artifact commands allow the management of artifacts within a minder control plane

```
minder artifact [flags]
```

### Options

```
  -h, --help   help for artifact
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
* [minder artifact get](minder_artifact_get.md)	 - Get artifact details
* [minder artifact list](minder_artifact_list.md)	 - List artifacts from a provider

