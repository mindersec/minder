## medic artifact list

List artifacts from a provider

### Synopsis

Artifact list will list artifacts from a provider

```
medic artifact list [flags]
```

### Options

```
  -g, --group-id int32    ID of the group for repo registration
  -h, --help              help for list
  -l, --limit int32       Number of repos to display per page (default 20)
  -o, --offset int32      Offset of the repos to display
  -f, --output string     Output format (json or yaml)
  -n, --provider string   Name for the provider to enroll
  -t, --type string       Type of artifact to list: npm, maven, rubygems, docker, nuget, container
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "localhost")
      --grpc-port int      Server port (default 8090)
```

### SEE ALSO

* [medic artifact](medic_artifact.md)	 - Manage repositories within a mediator control plane

