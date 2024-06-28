---
title: Adding Users to your Project
sidebar_position: 100
---

## Prerequisites

* The `minder` CLI application
* A Stacklok account

## Overview

To add a new user to your project, you need to:

1. Identify your project ID
2. Have the new user create a Minder account
3. Identify the new user's role
4. Add the user to your project

## Identify your project ID
Identify the project that you want to add a new user to. To see all the projects that are available to you, use the [`minder project list`](../ref/cli/minder_project_list) command.

```
+--------------------------------------+-------------------+
|                  ID                  |        NAME       |
+--------------------------------------+-------------------+
| 086df3e2-f1bb-4b3a-b2fe-0ebd147e1538 | my_minder_project |
+--------------------------------------+-------------------+
```

In this example, the `my_minder_project` project has Project ID `086df3e2-f1bb-4b3a-b2fe-0ebd147e1538`.

## Have the new user create a Minder account
To add a user to your project, that user must first [create their Minder account](https://docs.stacklok.com/minder/getting_started/login#logging-in-to-the-stacklok-hosted-instance), and provide you with their user ID.

The new user must create an account and log in using [`minder auth login`](../ref/cli/minder_auth_login). After login, the user ID will be displayed as the `Subject`. For example:

```
Here are your details:

+----------------+--------------------------------------+
|      KEY       |                VALUE                 |
+----------------+--------------------------------------+
| Subject        | ef5588e2-802b-47cb-b64a-52167acfea41 |
+----------------+--------------------------------------+
| Created At     | 2024-04-01 09:10:11.121314           |
|                | +0000 UTC                            |
+----------------+--------------------------------------+
...
```

In this example, the new user's User ID is `ef5588e2-802b-47cb-b64a-52167acfea41`. Once the new user has provided you with their User ID, you can add them to your project.

## Identify their role
When adding a user into your project, it's crucial to assign them the appropriate role based on their responsibilities and required access levels.

Roles are [documented here](user_roles). To view the available roles in your project, and their descriptions, run:

```
minder project role list
```

## Add the user to your project
To add a user to your project, follow these steps:

1. Determine the User's Role: Decide the appropriate role for the user based on their responsibilities.

2. Execute the Command:

   ```bash
   minder project role grant --project project-id --sub user-id --role desired-role
    ```

   - Replace `project-id` with the identifier of the project to which you want to add the user.
   - Replace `user-id` with the unique identifier of the user you want to add.
   - Replace `desired-role` with the chosen role for the user (e.g., `viewer`, `editor`).

You can then view all the project collaborators and their roles by executing:
```bash
minder project role grant list --project project-id
```
