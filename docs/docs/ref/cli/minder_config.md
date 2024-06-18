---
title: minder config
---
## minder config

How to manage minder CLI configuration

### Synopsis

In addition to the command-line flags, many minder options can be set via a configuration file in the YAML format.

Configuration options include:
- provider
- project
- output
- grpc_server.host
- grpc_server.port
- grpc_server.insecure
- identity.cli.issuer_url
- identity.cli.client_id

By default, we look for the file as $PWD/config.yaml and $XDG_CONFIG_PATH/minder/config.yaml. You can specify a custom path via the --config flag, or by setting the MINDER_CONFIG environment variable.

### Options

```
  -h, --help   help for config
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

