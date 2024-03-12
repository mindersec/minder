---
title: minder project
---
## minder project

Manage project within a minder control plane

### Synopsis

The minder project commands manage projects within a minder control plane.

```
minder project [flags]
```

### Options

```
  -h, --help   help for project
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.stacklok.com")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")
```

### SEE ALSO

* [minder](minder.md)	 - Minder controls the hosted minder service
* [minder project create](minder_project_create.md)	 - Create a sub-project within a minder control plane
* [minder project delete](minder_project_delete.md)	 - Delete a sub-project within a minder control plane
* [minder project list](minder_project_list.md)	 - List the projects available to you within a minder control plane
* [minder project role](minder_project_role.md)	 - Manage roles within a minder control plane

