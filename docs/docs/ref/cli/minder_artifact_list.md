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
      --from string         Filter artifacts from a source, example: from=repository=owner/repo
  -h, --help                help for list
  -f, --output string       Output format (json or yaml)
  -g, --project-id string   ID of the project for repo registration
  -p, --provider string     Name for the provider to enroll (default "github")
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

* [minder artifact](minder_artifact.md)	 - Manage artifacts within a minder control plane

