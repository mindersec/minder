## medic artifact get

Get artifact details

### Synopsis

Artifact get will get artifact details from an artifact, for a given ID

```
medic artifact get [flags]
```

### Options

```
  -h, --help                    help for get
  -i, --id string               ID of the artifact to get info from
  -v, --latest-versions int32   Latest artifact versions to retrieve (default 1)
      --tag string              Specific artifact tag to retrieve
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

* [medic artifact](medic_artifact.md)	 - Manage artifacts within a mediator control plane

