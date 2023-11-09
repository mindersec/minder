---
title: minder artifact get
---
## minder artifact get

Get artifact details

### Synopsis

Artifact get will get artifact details from an artifact, for a given ID

```
minder artifact get [flags]
```

### Options

```
  -h, --help                    help for get
  -i, --id string               ID of the artifact to get info from
  -v, --latest-versions int32   Latest artifact versions to retrieve (default 1)
      --tag string              Specific artifact tag to retrieve
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

* [minder artifact](minder_artifact.md)	 - Manage artifacts within a minder control plane

