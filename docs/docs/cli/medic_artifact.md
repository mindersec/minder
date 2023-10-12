## medic artifact

Manage artifacts within a mediator control plane

### Synopsis

The medic artifact commands allow the management of artifacts within a mediator control plane

```
medic artifact [flags]
```

### Options

```
  -h, --help   help for artifact
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
* [medic artifact get](medic_artifact_get.md)	 - Get artifact details
* [medic artifact list](medic_artifact_list.md)	 - List artifacts from a provider

