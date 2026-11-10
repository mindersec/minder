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

Minder authorizes the Docker Hub provider using an access token you supply.
Enroll the provider with:

```bash
minder provider enroll --class dockerhub --token <TOKEN> --provider-config dockerhub-config.json
```

See [Enrolling a provider](#enrolling-a-provider) for the namespace
configuration file.

There is no separate username field — only the token is collected and stored.
The token's shape isn't validated against Docker Hub at enrollment time; Minder
only checks that it's a non-empty string in the expected format.

Minder sends this token when it lists the repositories in your namespace
through the Docker Hub API (`hub.docker.com`). It does not send the token when
it calls the Docker Hub registry (`registry-1.docker.io`) to list tags, resolve
a tag to a digest, or fetch a manifest — those calls are made anonymously,
regardless of what you've configured. Because registering an image resolves its
digest this way, private Docker Hub repositories are not currently supported by
this provider.

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
