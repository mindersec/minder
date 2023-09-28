## Importing realms
Any JSON files in the `import` directory will be used to [import realms](https://www.keycloak.org/server/importExport) into the Keycloak instance.

The existing file `stacklok-realm-with-user-and-client.json` creates a new realm named stacklok.  

The stacklok realm:
1) defines the superadmin role
1) contains two clients, mediator-cli and mediator-ui
1) contains one user, with the superadmin role
1) customizes the mediator-cli login page to use Stacklok branding