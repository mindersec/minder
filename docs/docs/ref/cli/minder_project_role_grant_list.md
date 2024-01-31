---
title: minder project role grant list
---
## minder project role grant list

List role grants within a given project

### Synopsis

The minder project role grant list command lists all role grants
on a particular project.

```
minder project role grant list [flags]
```

### Options

```
  -h, --help            help for list
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
```

### SEE ALSO

* [minder project role grant](minder_project_role_grant.md)	 - Grant a role to a subject on a project within the minder control plane

