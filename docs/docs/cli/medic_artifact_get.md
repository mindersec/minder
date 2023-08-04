## medic artifact get

Get artifact details

### Synopsis

Artifact get will get artifact details from an artifact, for a given type and name

```
medic artifact get [flags]
```

### Options

```
  -g, --group-id int32          ID of the group for repo registration
  -h, --help                    help for get
  -v, --latest-versions int32   Latest artifact versions to retrieve (default 1)
  -n, --name string             Name of the artifact to get info from
  -p, --provider string         Name for the provider to enroll
      --tag string              Specific artifact tag to retrieve
  -t, --type string             Type of the artifact to get info from (npm, maven, rubygems, docker, nuget, container)
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "localhost")
      --grpc-port int      Server port (default 8090)
```

### SEE ALSO

* [medic artifact](medic_artifact.md)	 - Manage repositories within a mediator control plane

