## medic artifact get

Get artifact details

### Synopsis

Artifact get will get artifact details from an artifact, for a given id

```
medic artifact get [flags]
```

### Options

```
  -h, --help                    help for get
  -i, --id int32                ID of the artifact to get info from
  -v, --latest-versions int32   Latest artifact versions to retrieve (default 1)
      --tag string              Specific artifact tag to retrieve
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "localhost")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 8090)
```

### SEE ALSO

* [medic artifact](medic_artifact.md)	 - Manage repositories within a mediator control plane

