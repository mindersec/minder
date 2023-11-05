---
title: minder apply
---
## minder apply

Appy a configuration to a minder control plane

### Synopsis

The minder apply command applies a configuration to a minder control plane.

```
minder apply (-f FILENAME) [flags]
```

### Options

```
  -f, --file string   Path to the configuration file to apply or - for stdin
  -h, --help          help for apply
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

