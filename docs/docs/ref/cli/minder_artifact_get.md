---
title: minder artifact get
---
## minder artifact get

Get artifact details

### Synopsis

The artifact get subcommand will get artifact details from an artifact, for a given ID.

```
minder artifact get [flags]
```

### Options

```
  -h, --help            help for get
  -i, --id string       ID of the artifact to get info from
  -n, --name string     name of the artifact to get info from in the form repoOwner/repoName/artifactName
  -o, --output string   Output format (one of json,yaml,table) (default "table")
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.stacklok.com")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")
  -j, --project string           ID of the project
  -p, --provider string          Name of the provider, i.e. github
```

### SEE ALSO

* [minder artifact](minder_artifact.md)	 - Manage artifacts within a minder control plane

