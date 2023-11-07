# Installing Minder

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
4) OAuth2 Application: For GitHub integration, you will need to create a GitHub OAuth2 application to link user identities in Keycloak.

## Postgres Installation
Minder requires a dedicated Postgres database to store its operational data. The database must have a dedicated user with the necessary privileges and credentials.

### Best Practices for Database Deployment
It is recommended to use two distinct database users:

- One for the Minder server operations.
- Another solely for database migrations.

You can find our database migration scripts at [Minder Database Migrations](https://github.com/stacklok/minder/tree/main/database/migrations).

## Ingress Configuration
Your ingress controller must be capable of handling both gRPC and HTTP/1 protocols. Please note that HTTP/2 compatibility has not been tested and is not guaranteed.

Minder exposes both HTTP and gRPC APIs, and our default Helm chart configuration enables ingress for both protocols. If your ingress solution requires different settings, please disable the default ingress in the Helm chart and configure it manually to meet your environment's needs.

## GitHub OAuth Application
For Minder to interact with GitHub repositories, a GitHub OAuth2 application is required. This is essential for Minder's operation, as it will use this application to authenticate and perform actions on GitHub repositories.

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
| aws.accountID | string | `"123456789012"` | AWS account ID where the service will be deployed |
| aws.migrate.iamRole | string | `"minder_migrate_role"` | IAM role for the migration operations in AWS |
| aws.server.iamRole | string | `"minder_server_role"` | IAM role for the server operations in AWS |
| db.host | string | `"postgres.postgres"` | Hostname for the database where Minder will store its data |
| deploymentSettings.extraVolumeMounts | string | `nil` | Additional volume mounts for the deployment |
| deploymentSettings.extraVolumes | string | `nil` | Additional volumes to mount into the deployment |
| deploymentSettings.image | string | `"ko://github.com/stacklok/minder/cmd/server"` | Image to use for the main Minder deployment |
| deploymentSettings.imagePullPolicy | string | `"IfNotPresent"` | Image pull policy for the main deployment |
| deploymentSettings.resources | object | `{"limits":{"cpu":4,"memory":"1.5Gi"},"requests":{"cpu":1,"memory":"1Gi"}}` | Compute resource requests and limits for the main deployment |
| deploymentSettings.secrets | object | `{"appSecretName":"minder-github-secrets","authSecretName":"minder-auth-secrets","identitySecretName":"minder-identity-secrets"}` | Names of the secrets for various Minder components |
| hostname | string | `"minder.example.com"` | The hostname for the Minder service |
| hpaSettings.maxReplicas | int | `1` | Maximum number of replicas for HPA |
| hpaSettings.metrics | object | `{"cpu":{"targetAverageUtilization":60}}` | Target CPU utilization percentage for HPA to scale |
| hpaSettings.minReplicas | int | `1` | Minimum number of replicas for HPA |
| ingress.annotations | object | `{}` | Ingress annotations |
| migrationSettings.image | string | `"ko://github.com/stacklok/minder/cmd/server"` | Image to use for the migration jobs |
| migrationSettings.imagePullPolicy | string | `"IfNotPresent"` | Image pull policy for the migration jobs |
| migrationSettings.resources | object | `{"limits":{"cpu":1,"memory":"300Mi"},"requests":{"cpu":"200m","memory":"200Mi"}}` | Compute resource requests and limits for the migration jobs |
| service.grpcPort | int | `8090` | GRPC port for the service to listen on |
| service.httpPort | int | `8080` | HTTP port for the service to listen on |
| service.metricPort | int | `9090` | Metrics port for the service to expose metrics on |
| serviceAccounts.migrate | string | `""` | ServiceAccount to be used for migration. If set, Minder will use this named ServiceAccount. |
| serviceAccounts.server | string | `""` | ServiceAccount to be used by the server. If set, Minder will use this named ServiceAccount. |
| trusty.endpoint | string | `"http://pi.pi:8000"` | Endpoint for the trusty service which Minder communicates with |

