---
title: minder project role grant
---
## minder project role grant

Grant a role to a subject on a project within the minder control plane

### Synopsis

The minder project role grant command allows one to grant a role
to a user (subject) on a particular project.

```
minder project role grant [flags]
```

### Options

```
  -e, --email string    email to send invitation to
  -h, --help            help for grant
  -o, --output string   Output format (one of json,yaml,table) (default "table")
  -r, --role string     the role to grant
  -s, --sub string      subject to grant access to
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

* [minder project role](minder_project_role.md)	 - Manage roles within a minder control plane
* [minder project role grant list](minder_project_role_grant_list.md)	 - List role grants within a given project

