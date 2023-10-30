## medic repo get

Get repository in the mediator control plane

### Synopsis

Repo get is used to get a repo with the mediator control plane

```
medic repo get [flags]
```

### Options

```
  -h, --help              help for get
  -n, --name string       Name of the repository (owner/name format)
  -f, --output string     Output format (json or yaml)
  -p, --provider string   Name for the provider to enroll
  -r, --repo-id string    ID of the repo to query
  -s, --status            Only return the status of the profiles associated to this repo
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

* [medic repo](medic_repo.md)	 - Manage repositories within a mediator control plane

