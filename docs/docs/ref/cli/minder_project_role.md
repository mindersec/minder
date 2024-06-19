---
title: minder project role
---
## minder project role

Manage roles within a minder control plane

### Synopsis

The minder role commands manage permissions within a minder control plane.

```
minder project role [flags]
```

### Options

```
  -h, --help             help for role
  -j, --project string   ID of the project
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

* [minder project](minder_project.md)	 - Manage project within a minder control plane
* [minder project role deny](minder_project_role_deny.md)	 - Deny a role to a subject on a project within the minder control plane
* [minder project role grant](minder_project_role_grant.md)	 - Grant a role to a subject on a project within the minder control plane
* [minder project role list](minder_project_role_list.md)	 - List roles on a project within the minder control plane
* [minder project role update](minder_project_role_update.md)	 - update a role to a subject on a project

