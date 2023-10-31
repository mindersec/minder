---
title: minder artifact list
---
## minder artifact list

List artifacts from a provider

### Synopsis

Artifact list will list artifacts from a provider

```
minder artifact list [flags]
```

### Options

```
  -h, --help                help for list
  -f, --output string       Output format (json or yaml)
  -g, --project-id string   ID of the project for repo registration
  -n, --provider string     Name for the provider to enroll
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "staging.stacklok.dev")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-realm string    Identity server realm (default "stacklok")
      --identity-url string      Identity server issuer URL (default "https://auth.staging.stacklok.dev")
```

### SEE ALSO

* [minder artifact](minder_artifact.md)	 - Manage artifacts within a minder control plane

