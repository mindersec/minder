---
title: medic profile
---
## medic profile

Manage profiles within a mediator control plane

### Synopsis

The medic profile subcommands allows the management of profiles within
a mediator controlplane.

```
medic profile [flags]
```

### Options

```
  -h, --help   help for profile
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
* [medic profile create](medic_profile_create.md)	 - Create a profile within a mediator control plane
* [medic profile delete](medic_profile_delete.md)	 - Delete a profile within a mediator control plane
* [medic profile get](medic_profile_get.md)	 - Get details for a profile within a mediator control plane
* [medic profile list](medic_profile_list.md)	 - List profiles within a mediator control plane

