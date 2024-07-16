---
title: Adding Users to your Project
sidebar_position: 100
---

## Prerequisites

- The `minder` CLI application
- A Minder account with [`admin` permission](../user_management/user_roles.md)

## Overview

To invite a new user to your project, you need to:

1. Identify the new user's role
2. Invite the user to your project
3. Have the user exercise the invitation code

## Identify their role

When adding a user into your project, it's crucial to assign them the
appropriate role based on their responsibilities and required access levels.

Roles are [documented here](user_roles.md). To view the available roles in your
project, and their descriptions, run:

```
minder project role list
```

## Invite the user to your project

Minder uses an invitation code system to allow users to accept (or decline) the
invitation to join a project. The invitation code may be delivered over email,
or may be copied into a channel like Slack or a ticket system. **Any user with
access to an unused invitation code may accept the invitation, so treat the code
like a password.**

To add a user to your project, follow these steps:

1. Determine the User's Role: Decide the appropriate role for the user based on
   their responsibilities.

2. Execute the Command:

   ```bash
   minder project role grant --email email@example.com --role desired-role
   ```

   - Replace `email@example.com` with the email address of the user you want to
     invite.
   - Replace `desired-role` with the chosen role for the user (e.g., `viewer`,
     `editor`).

3. You will receive a response message including an invitation code which can be
   used with
   [`minder auth invite accept`](../ref/cli/minder_auth_invite_accept.md).

## Have the User Exercise the Invitation Code

Relay the invitation code to the user who you are inviting to join your project.
They will need to [install the `minder` CLI](../getting_started/install_cli.md)
and run `minder auth invite accept <invitation code>` to accept the invitation.
Invitations will expire when used, or after 7 days, though users who have not
accepted an invitation can be invited again.

## Viewing outstanding invitations

You can then view all the project collaborators and outstanding user invitations
by executing:

```bash
minder project role grant list
```

## Working with Multiple Projects

When you have access to more than one project, you will need to qualify many
`minder` commands with which project you want them to apply to. You can either
use the `--project` flag, set a default in your
[minder configuration](../ref/cli_configuration.md), or set the `MINDER_PROJECT`
environment variable. To see all the projects that are available to you, use the
[`minder project list`](../ref/cli/minder_project_list.md) command.

```
+--------------------------------------+-------------------+
|                  ID                  |        NAME       |
+--------------------------------------+-------------------+
| 086df3e2-f1bb-4b3a-b2fe-0ebd147e1538 | my_minder_project |
| f9f4aef0-74af-4909-a0c3-0e8ac7fbc38d | another_project   |
+--------------------------------------+-------------------+
```

In this example, the `my_minder_project` project has Project ID
`086df3e2-f1bb-4b3a-b2fe-0ebd147e1538`, and `another_project` has ID
`f9f4aef0-74af-4909-a0c3-0e8ac7fbc38d`. Note that you need to specify the
project _ID_ when using the `--project` flag or `MINDER_CONFIG`, not the project
name.
