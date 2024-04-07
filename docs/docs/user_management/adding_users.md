# Adding users to a project

## Prerequisites

* The `minder` CLI application
* A Stacklok account

## Roles Overview
When incorporating a user into your project, it's crucial to assign them the appropriate role based on their responsibilities and required access levels. 
Minder currently offers the following roles:

- `viewer`: Provides read-only access to the project. Users with this role can view associated resources such as enrolled repositories, rule types, and profiles.
- `editor`: Grants the same permissions as the viewer role, along with the ability to edit project resources, excluding the project itself and the list of providers.
- `admin`: Grants administrator rights on the project. Users with this role have the same permissions as editor and can also modify the project and associated providers.
- `policy_writer`: Allows users to create rule types and profiles.
- `permissions_manager`: Allows users to manage roles for other users within the project.

You can also view the available roles and their descriptions by executing:
```bash
minder project role list
```

## Adding a user
To add a user to your project, follow these steps:

1) Determine the User's Role: Decide the appropriate role for the user based on their responsibilities.

2) Execute the Command:
 ```bash
 minder project role grant --sub user-id --role desired-role --project project-id
 ```
 - Replace `user-id` with the unique identifier of the user you want to add.
 - Replace `desired-role` with the chosen role for the user (e.g., `viewer`, `editor`).
 - Replace `project-id` with the identifier of the project to which you want to add the user.

You can then view all the project collaborators and their roles by executing:
```bash
minder project role grant list --project project-id
```
