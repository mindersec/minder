# User roles in Minder

When incorporating a user into your project, it's crucial to assign them the appropriate role based on their responsibilities and required access levels.
Minder currently offers the following roles:

- `viewer`: Provides read-only access to the project. Users with this role can view associated resources such as enrolled repositories, rule types, profiles and the status of rule evaluations.
- `editor`: In addition to the viewer permissions, editors can author profiles and rule types, as well as add resources to manage. Editors cannot enroll additional providers or change or delete projects.
- `admin`: Admins have full permissions on the project. In addition to the editor permissions, users with this role can modify the project, enroll additional providers, and manage roles for other users within the project.
- `policy_writer`: Allows users to create rule types and profiles. Unlike editors, policy writers cannot add or remove resources from the project.
- `permissions_manager`: Allows users to manage roles for other users within the project.
