## medic provider

Manage providers within a mediator control plane

### Synopsis

The medic provider commands manage providers within a mediator control plane.

```
medic provider [flags]
```

### Options

```
  -h, --help   help for provider
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
* [medic provider enroll](medic_provider_enroll.md)	 - Enroll a provider within the mediator control plane

