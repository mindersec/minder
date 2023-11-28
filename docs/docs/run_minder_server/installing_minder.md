---
sidebar_label: Helm Install
sidebar_position: 80
---

# Installing Minder with Helm

## Keycloak Installation
Minder is designed to operate without storing user credentials or personal information. To achieve this, it relies on an external identity provider. While Minder is compatible with any OpenID Connect (OIDC)-enabled identity provider, we have thoroughly tested it with Keycloak and thus recommend it for a seamless integration.

### Getting Started with Keycloak
To install Keycloak as your identity provider, please refer to the following resources for detailed instructions:

- Keycloak Operator Installation Guide: [Keycloak Operator Installation](https://www.keycloak.org/operator/installation)
- Keycloak Tutorials for Beginners: [Keycloak Tutorials](https://keycloak.ch/keycloak-tutorials/tutorial-1-installing-and-running-keycloak/)

After the installation of Keycloak, there are specific settings and configurations required for Minder to function properly:

1) **Realm Configuration:** Set up a dedicated realm in Keycloak for Minder's use.
2) **Client Setup:** Create two separate clients within the realm:
    - **minder-cli:** A client for command-line interactions.
    - **minder-server:** A client for server-side operations.
3) Identity Provider Linkage: Connect your chosen Identity Provider (e.g., GitHub, Google) to Keycloak. To facilitate this process, you may use the initialization script available at [Minder Identity Initialization Script](https://github.com/stacklok/minder/blob/main/identity/scripts/initialize.sh).

## Postgres Installation
Minder requires a dedicated Postgres database to store its operational data. The database must have a dedicated user with the necessary privileges and credentials.

### Best Practices for Database Deployment
It is recommended to use two distinct database users:

- One for the Minder server operations.
- Another solely for database migrations.

You can find our database migration scripts at [Minder Database Migrations](https://github.com/stacklok/minder/tree/main/database/migrations).

## Ingress Configuration
Your ingress controller must be capable of handling both gRPC and HTTP/1 protocols.

Minder exposes both HTTP and gRPC APIs, and our default Helm chart configuration enables ingress for both protocols. If your ingress solution requires different settings, please disable the default ingress in the Helm chart and configure it manually to meet your environment's needs.

## GitHub OAuth Application
For Minder to interact with GitHub repositories, a GitHub OAuth2 application is required. This is essential for Minder's operation, as it will use this application to authenticate and perform actions on GitHub repositories.

Please ensure the following secrets are securely stored and handled, as they contain sensitive information crucial for the authentication and operation of Minder's integrations:

- **minder-identity-secrets:** a secret with the key identity_client_secret and the value being the keycloak minder-server client secret.
- **minder-auth-secrets:** a secret with the key token_key_passphrase and unique content, used to encrypt tokens in the database.
- **minder-github-secrets:** a secret with the keys client_id and client_secret that contains the GitHub OAuth app secrets.

## Helm Chart Parameters
### Minder

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2023-07-31](https://img.shields.io/badge/AppVersion-2023--07--31-informational?style=flat-square)

Deploy Minder on Kubernetes

### Requirements

| Repository | Name | Version |
|------------|------|---------|
| oci://registry-1.docker.io/bitnamicharts | common | 2.x.x |

### Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| aws.accountID | string | `"123456789012"` |  |
| aws.migrate | object, optional | `{"iamRole":"minder_migrate_role"}` | AWS IAM migration settings |
| aws.migrate.iamRole | string | `"minder_migrate_role"` | IAM role to use for the migration job |
| aws.server | object, optional | `{"iamRole":"minder_server_role"}` | AWS IAM server settings |
| aws.server.iamRole | string | `"minder_server_role"` | IAM role to use for the server |
| db.host | string | `"postgres.postgres"` | database host to use |
| deploymentSettings.extraVolumeMounts | array, optional | `nil` | Additional volume mounts |
| deploymentSettings.extraVolumes | array, optional | `nil` | Additional volumes to mount |
| deploymentSettings.image | string | `"ko://github.com/stacklok/minder/cmd/server"` | image to use for the main deployment |
| deploymentSettings.imagePullPolicy | string | `"IfNotPresent"` | image pull policy to use for the main deployment |
| deploymentSettings.resources | object | `{"limits":{"cpu":4,"memory":"1.5Gi"},"requests":{"cpu":1,"memory":"1Gi"}}` | resources to use for the main deployment |
| deploymentSettings.secrets.appSecretName | string | `"minder-github-secrets"` | name of the secret containing the github configuration |
| deploymentSettings.secrets.authSecretName | string | `"minder-auth-secrets"` | name of the secret containing the auth configuration |
| deploymentSettings.secrets.dbQueueSecretName | string | `"minder-db-queue-secrets"` | name of the secret containing the database queue configuration |
| deploymentSettings.secrets.identitySecretName | string | `"minder-identity-secrets"` | name of the secret containing the identity configuration |
| deploymentSettings.sidecarContainers | array, optional | `nil` | Additional configuration for sidecar containers |
| extra_config | string | `"# Add content here\n"` | Additional configuration yaml beyond what's in config.yaml.example |
| extra_config_migrate | string | `"# Add even more content here\n"` | Additional configuration yaml that's applied to the migration job |
| hostname | string | `"minder.example.com"` | hostname to ue for the ingress configuration |
| hpaSettings.maxReplicas | int | `1` | maximum number of replicas for the HPA |
| hpaSettings.metrics | object | `{"cpu":{"targetAverageUtilization":60}}` | metrics to use for the HPA |
| hpaSettings.minReplicas | int | `1` | minimum number of replicas for the HPA |
| ingress.annotations | object, optional | `{}` | annotations to use for the ingress |
| migrationSettings.extraVolumeMounts | array, optional | `nil` | Additional volume mounts |
| migrationSettings.extraVolumes | array, optional | `nil` | Additional volumes to mount |
| migrationSettings.image | string | `"ko://github.com/stacklok/minder/cmd/server"` | image to use for the migration job |
| migrationSettings.imagePullPolicy | string | `"IfNotPresent"` | image pull policy to use for the migration job |
| migrationSettings.resources | object | `{"limits":{"cpu":1,"memory":"300Mi"},"requests":{"cpu":"200m","memory":"200Mi"}}` | resources to use for the migration job |
| migrationSettings.sidecarContainers | array, optional | `nil` | Additional configuration for sidecar containers |
| service.grpcPort | int | `8090` | port for the gRPC API |
| service.httpPort | int | `8080` | port for the HTTP API |
| service.metricPort | int | `9090` | port for the metrics endpoint |
| serviceAccounts.migrate | string, optional | `""` | If non-empty, minder will use the named ServiceAccount resources rather than creating a ServiceAccount |
| serviceAccounts.server | string, optional | `""` | If non-empty, minder will use the named ServiceAccount resources rather than creating a ServiceAccount |
| trusty.endpoint | string | `"http://pi.pi:8000"` | trusty host to use |
