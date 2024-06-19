---
title: minder set-project
---
## minder set-project

Move the current context to another project

### Synopsis

The minder set-project command moves the current context to another project.
Passing a UUID will move the context to the project with that UUID. This is akin to
using an absolute path in a filesystem.

```
minder set-project [flags]
```

### Options

```
  -h, --help   help for set-project
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

* [minder](minder.md)	 - Minder controls the hosted minder service

