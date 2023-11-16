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

  - `minder-auth-secrets`: Needs to contain public and private keys for access
    and refresh tokens. The keys must be named as follows:

    - `access_token_rsa`, `access_token_rsa.pub`
    - `refresh_token_rsa`, `refresh_token_rsa.pub`

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
| aws.migrate.iamRole | string | `"minder_migrate_role"` |  |
| aws.server.iamRole | string | `"minder_server_role"` |  |
| db.host | string | `"postgres.postgres"` |  |
| deploymentSettings.extraVolumeMounts | string | `nil` |  |
| deploymentSettings.extraVolumes | string | `nil` |  |
| deploymentSettings.image | string | `"ko://github.com/stacklok/minder/cmd/server"` |  |
| deploymentSettings.imagePullPolicy | string | `"IfNotPresent"` |  |
| deploymentSettings.resources.limits.cpu | int | `4` |  |
| deploymentSettings.resources.limits.memory | string | `"1.5Gi"` |  |
| deploymentSettings.resources.requests.cpu | int | `1` |  |
| deploymentSettings.resources.requests.memory | string | `"1Gi"` |  |
| deploymentSettings.secrets.appSecretName | string | `"minder-github-secrets"` |  |
| deploymentSettings.secrets.authSecretName | string | `"minder-auth-secrets"` |  |
| deploymentSettings.secrets.identitySecretName | string | `"minder-identity-secrets"` |  |
| deploymentSettings.sidecarContainers | string | `nil` |  |
| extra_config | string | `"# Add content here\n"` |  |
| extra_config_migrate | string | `"# Add even more content here\n"` |  |
| hostname | string | `"minder.example.com"` |  |
| hpaSettings.maxReplicas | int | `1` |  |
| hpaSettings.metrics.cpu.targetAverageUtilization | int | `60` |  |
| hpaSettings.minReplicas | int | `1` |  |
| ingress.annotations | object | `{}` |  |
| migrationSettings.extraVolumeMounts | string | `nil` |  |
| migrationSettings.extraVolumes | string | `nil` |  |
| migrationSettings.image | string | `"ko://github.com/stacklok/minder/cmd/server"` |  |
| migrationSettings.imagePullPolicy | string | `"IfNotPresent"` |  |
| migrationSettings.resources.limits.cpu | int | `1` |  |
| migrationSettings.resources.limits.memory | string | `"300Mi"` |  |
| migrationSettings.resources.requests.cpu | string | `"200m"` |  |
| migrationSettings.resources.requests.memory | string | `"200Mi"` |  |
| migrationSettings.sidecarContainers | string | `nil` |  |
| service.grpcPort | int | `8090` |  |
| service.httpPort | int | `8080` |  |
| service.metricPort | int | `9090` |  |
| serviceAccounts.migrate | string | `""` |  |
| serviceAccounts.server | string | `""` |  |
| trusty.endpoint | string | `"http://pi.pi:8000"` |  |