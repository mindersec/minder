---
title: minder project role update
---
## minder project role update

update a role to a subject on a project

### Synopsis

The minder project role update command allows one to update a role
to a user (subject) on a particular project.

```
minder project role update [flags]
```

### Options

```
  -e, --email string   email to send invitation to
  -h, --help           help for update
  -r, --role string    the role to update it to
  -s, --sub string     subject to update role access for
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

