---
title: Projects
sidebar_position: 20
---

# Projects in Minder

Projects in Minder are a way to group entities together, and to share access
with other people in your organization. They are a way to organize your entities
(repositories, artifacts, etc.) based on the policy you want to enforce or the
team that owns them.

Each user in an project is [assigned a role](../user_management/user_roles.md),
such as `editor` or `admin`. Projects must have at least one administrator, who
has permission to
[invite other users to the project](../user_management/adding_users.md) and
change users roles within the project.

When creating an account, Minder will automatically create a default project for
you, unless you are accepting an invitation to an existing project. You can
[create additional projects](../how-to/create_project.md) as a way to organize
and secure your entities, and manage access for your team members.

If you are a member (have a role in) more than one project, you can use
[the `MINDER_PROJECT` environment variable](../ref/cli_configuration.md) or the
`--project` flag to select a specific project to operate on with the CLI. If you
have access to multiple projects and no project is selected, Minder will report
an error asking you to select a project.
