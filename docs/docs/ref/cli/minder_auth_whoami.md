---
title: minder auth whoami
---
## minder auth whoami

whoami for current user

### Synopsis

whoami gets information about the current user from the minder server

```
minder auth whoami [flags]
```

### Options

```
  -h, --help            help for whoami
  -o, --output string   Output format (one of json,yaml,table) (default "table")
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

* [minder auth](minder_auth.md)	 - Authorize and manage accounts within a minder control plane

