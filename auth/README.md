# About the model

We use per-resource-type permissions off of "project" because
we don't allow granting permissions on individual resources, only
on projects.  This allows us to minimize the amount of state we
need to keep consistent between OpenFGA and the main database.
