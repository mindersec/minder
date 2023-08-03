# Helm charts for mediator server

(These are a work in progress)

Stacklok runs mediator on Kubernetes, using these helm charts. In order to use
these helm charts, you will need the following:

- Postgres running in your cluster. In particular, you need a `postgres` Service
  in the `postgres` Namespace. The [`k8s-dev-setup`](../k8s-dev-setup/)
  directory has a sample postgres for local development and testing purposes.
  Note that the Postgres in that setup is **NOT** durable.

- The following Kubernetes secrets:

  - `mediator-auth-secrets`: Needs to contain public and private keys for access
    and refresh tokens. The keys must be named as follows:

    - `access_token_rsa`, `access_token_rsa.pub`
    - `refresh_token_rsa`, `refresh_token_rsa.pub`

  - `mediator-github-secrets`: Needs to contain API credentials for a GitHub
    app. In particular, the following keys are required:
    - `client_id`: The GitHub client ID to be used by Mediator
    - `client_secret`: The GitHub client secret to be used by Mediator

- In addition, if you are using Mediator images which require authentication,
  you will want to create a `docker-registry` type credential with the name
  `mediator-pull-secret`

## Building and running

You can build a (local) helm chart with `make helm` at the top-level of the
Mediator repository. You can then run it with:

```helm
helm install mediator config/helm/mediator-0.1.0.tgz
```

Note that the helm chart does not specify a namespace, so mediator will be
installed in the namespace specified by your current Kubernetes context.
