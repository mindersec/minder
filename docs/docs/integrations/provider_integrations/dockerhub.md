---
title: Docker Hub provider
sidebar_label: Docker Hub
sidebar_position: 40
---

A provider connects Minder to your software supply chain. It lets Minder know
where to look for your repositories, artifacts, and other entities, in order to
make them available for registration. It also tells Minder how to interact with
your supply chain to enable features such as alerting and remediation. Finally,
it handles the way Minder authenticates to the external service.

## Authorization methods

Minder authorizes the Docker Hub provider using an access token you supply,
similar to the Personal Access Token (PAT) method for GitLab.

## Prerequisites

Before enrolling Docker Hub as a provider, you will need:

- The `dockerhub_provider` feature flag enabled on your Minder server (see
  [Using feature flags](../../developer_guide/feature_flags)).
- The Docker Hub **namespace** whose images you want Minder to monitor. This is
  the organization or user name that appears before the image name, such as
  `library` in `library/alpine`.
- A Docker Hub access token with read access to that namespace. See the
  [Docker Hub access tokens documentation](https://docs.docker.com/security/access-tokens/)
  for how to create one.

## Enrolling a provider

The Docker Hub provider needs to know which namespace to use. Create a provider
configuration file, for example `dockerhub-config.json`, with your namespace:

```json
{
  "dockerhub": {
    "namespace": "<namespace>"
  }
}
```

Then enroll the provider, passing your access token with `--token` and the
configuration file with `--provider-config`:

```bash
minder provider enroll --class dockerhub --token <access-token> --provider-config dockerhub-config.json
```

The configuration file must be JSON.

Once enrolled, you can register images from your namespace. See
[Register container images](../../how-to/register_container_images.md).
