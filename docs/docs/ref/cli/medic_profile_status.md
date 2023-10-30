---
title: medic profile status
---
## medic profile_status

Manage profile status within a mediator control plane

### Synopsis

The medic profile_status subcommands allows the management of profile status within
a mediator controlplane.

```
medic profile_status [flags]
```

### Options

```
  -h, --help   help for profile_status
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "staging.stacklok.dev")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "mediator-cli")
      --identity-realm string    Identity server realm (default "stacklok")
      --identity-url string      Identity server issuer URL (default "https://auth.staging.stacklok.dev")
```

### SEE ALSO

* [medic](medic.md)	 - Medic controls mediator via the control plane
* [medic profile_status get](medic_profile_status_get.md)	 - Get profile status within a mediator control plane
* [medic profile_status list](medic_profile_status_list.md)	 - List profile status within a mediator control plane

