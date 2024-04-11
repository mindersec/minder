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
  -c, --class string   Provider class, defaults to github-app (default "github-app")
  -h, --help           help for enroll
  -o, --owner string   Owner to filter on for provider resources (Legacy GitHub only)
      --skip-browser   Skip opening the browser for OAuth flow
  -t, --token string   Personal Access Token (PAT) to use for enrollment (Legacy GitHub only)
  -y, --yes            Bypass any yes/no prompts when enrolling a new provider
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
  -p, --provider class           DEPRECATED - use class flag of `enroll` instead
```

### SEE ALSO

* [minder provider](minder_provider.md)	 - Manage providers within a minder control plane

