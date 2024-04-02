# Helm charts for Minder server

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=for-the-badge)
![Type: application](https://img.shields.io/badge/Type-application-informational?style=for-the-badge)
![AppVersion: 2023-07-31](https://img.shields.io/badge/AppVersion-2023--07--31-informational?style=for-the-badge)

Deploy Minder on Kubernetes

Stacklok runs Minder on Kubernetes, using these helm charts. In order to use
these helm charts, you will need the following:

- Postgres running in your cluster. In particular, you need a `postgres` Service
  in the `postgres` Namespace. The [`k8s-dev-setup`](../k8s-dev-setup/)
  directory has a sample postgres for local development and testing purposes.
  Note that the Postgres in that setup is **NOT** durable.

- The following Kubernetes secrets:

  - `minder-github-secrets`: Needs to contain API credentials for a GitHub
    app. In particular, the following keys are required:
    - `client_id`: The GitHub client ID to be used by Minder
    - `client_secret`: The GitHub client secret to be used by Minder

  - `minder-identity-secrets`: Needs to contain the OAuth 2 client secret for Minder
    server when authenticating with Keycloak. In particular, the following keys are required:
    - `identity_client_secret`: The Keycloak client secret to be used by Minder server

- In addition, if you are using Minder images which require authentication,
  you will want to create a `docker-registry` type credential with the name
  `minder-pull-secret`

- In addition, if you are using a GitHub App for authorization, you will need:
    - `minder-github-app-secrets`: Needs to contain API credentials for a GitHub
      app. In particular, the following keys are required:
        - `client_id`: The GitHub client ID to be used by Minder
        - `client_secret`: The GitHub client secret to be used by Minder
        - `private_key`: The GitHub App's private key for minting JWTs

## Building and running

You can build a (local) helm chart with `make helm` at the top-level of the
Minder repository. You can then run it with:

```helm
helm install minder config/helm/minder-0.1.0.tgz
```

Note that the helm chart does not specify a namespace, so Minder will be
installed in the namespace specified by your current Kubernetes context.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| aws.accountID | string | `"123456789012"` |  |
| aws.migrate | object, optional | `{"iamRole":"minder_migrate_role"}` | AWS IAM migration settings |
| aws.migrate.iamRole | string | `"minder_migrate_role"` | IAM role to use for the migration job |
| aws.server | object, optional | `{"iamRole":"minder_server_role"}` | AWS IAM server settings |
| aws.server.iamRole | string | `"minder_server_role"` | IAM role to use for the server |
| db.host | string | `"postgres.postgres"` | Database host to use |
| deploymentSettings.extraVolumeMounts | array, optional | `nil` | Additional volume mounts |
| deploymentSettings.extraVolumes | array, optional | `nil` | Additional volumes to mount |
| deploymentSettings.image | string | `"ko://github.com/stacklok/minder/cmd/server"` | Image to use for the main deployment |
| deploymentSettings.imagePullPolicy | string | `"IfNotPresent"` | Image pull policy to use for the main deployment |
| deploymentSettings.initContainers | array, optional | `nil` | Additional init containers to run |
| deploymentSettings.resources | object | `{"limits":{"cpu":4,"memory":"1.5Gi"},"requests":{"cpu":1,"memory":"1Gi"}}` | Resources to use for the main deployment |
| deploymentSettings.secrets.appSecretName | string | `"minder-github-secrets"` | Name of the secret containing the GitHub configuration |
| deploymentSettings.secrets.authSecretName | string | `"minder-auth-secrets"` | Name of the secret containing the auth configuration |
| deploymentSettings.secrets.githubAppSecretName | string | `"minder-github-app-secrets"` | Name of the secret containing the GitHub App configuration |
| deploymentSettings.secrets.identitySecretName | string | `"minder-identity-secrets"` | Name of the secret containing the identity configuration |
| deploymentSettings.sidecarContainers | array, optional | `nil` | Additional configuration for sidecar containers |
| deploymentSettings.terminationGracePeriodSeconds | int | `30` | Termination grace period for the main deployment |
| extra_config | string | `"# Add content here\n"` | Additional configuration yaml beyond what's in server-config.yaml.example |
| extra_config_migrate | string | `"# Add even more content here\n"` | Additional configuration yaml that's applied to the migration job |
| hostname | string | `"minder.example.com"` | Hostname to use for the ingress configuration |
| hpaSettings.maxReplicas | int | `1` | Maximum number of replicas for the HPA |
| hpaSettings.metrics | object | `{"cpu":{"targetAverageUtilization":60}}` | Metrics to use for the HPA |
| hpaSettings.minReplicas | int | `1` | Minimum number of replicas for the HPA |
| ingress.annotations | object, optional | `{}` | annotations to use for the ingress |
| migrationSettings.extraVolumeMounts | array, optional | `nil` | Additional volume mounts |
| migrationSettings.extraVolumes | array, optional | `nil` | Additional volumes to mount |
| migrationSettings.image | string | `"ko://github.com/stacklok/minder/cmd/server"` | Image to use for the migration job |
| migrationSettings.imagePullPolicy | string | `"IfNotPresent"` | Image pull policy to use for the migration job |
| migrationSettings.resources | object | `{"limits":{"cpu":1,"memory":"300Mi"},"requests":{"cpu":"200m","memory":"200Mi"}}` | Resources to use for the migration job |
| migrationSettings.sidecarContainers | array, optional | `nil` | Additional configuration for sidecar containers |
| service.grpcPort | int | `8090` | Port for the gRPC API |
| service.httpPort | int | `8080` | Port for the HTTP API |
| service.metricPort | int | `9090` | Port for the metrics endpoint |
| serviceAccounts.migrate | string, optional | `""` | If non-empty, minder will use the named ServiceAccount resources rather than creating a ServiceAccount |
| serviceAccounts.server | string, optional | `""` | If non-empty, minder will use the named ServiceAccount resources rather than creating a ServiceAccount |
| trusty.endpoint | string | `"https://api.trustypkg.dev"` | Trusty host to use |