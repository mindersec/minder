---
title: minder provider enroll
---
## minder provider enroll

Enroll a provider within the minder control plane

### Synopsis

The minder provider enroll command allows a user to enroll a provider
such as GitHub into the minder control plane. Once enrolled, users can perform
actions such as adding repositories.

```
minder provider enroll [flags]
```

### Options

```
  -h, --help           help for enroll
  -o, --owner string   Owner to filter on for provider resources
      --skip-browser   Skip opening the browser for OAuth flow
  -t, --token string   Personal Access Token (PAT) to use for enrollment
  -y, --yes            Bypass yes/no prompt when enrolling new provider
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
  -p, --provider string          Name of the provider, i.e. github (default "github")
```

### SEE ALSO

* [minder provider](minder_provider.md)	 - Manage providers within a minder control plane

