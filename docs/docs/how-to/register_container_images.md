---
title: Register container images
sidebar_position: 52
---

Once you have enrolled a Docker Hub or Quay.io provider, you can register
container images with your Minder project. Registered images are artifacts that
your profiles can apply to.

## Prerequisites

Before you can register an image, you must
[enroll a provider](../getting_started/enroll_provider) for the registry that
hosts it:

- [Docker Hub](../integrations/provider_integrations/dockerhub.md)
- [Quay.io](../integrations/provider_integrations/quay.md)

Each provider is enrolled with a namespace, such as a Docker Hub organization or
a Quay.io organization. You can register images from that namespace.

## Register an image

To register an image, run:

```bash
minder entity register --provider <provider-name> --type artifact --property name=<image>:<tag>
```

The `name` property is the image's repository name and tag **within the
provider's namespace**. Don't include the registry host or the namespace.

For example, with a Docker Hub provider named `dockerhub-library` that is
enrolled with the `library` namespace, this registers
`docker.io/library/alpine:latest`:

```bash
minder entity register --provider dockerhub-library --type artifact --property name=alpine:latest
```

Always include the tag. Each registered artifact is one tag of an image, so to
register more than one tag, run the command once for each tag.

When registration succeeds, Minder prints the ID of the new entity:

```text
Successfully registered entity with ID: <entity-id>
```

## Check your registered images

To list the images registered for a provider, run:

```bash
minder entity list --provider <provider-name> --type artifact
```

The table shows each artifact's name, provider and ID.

## Next steps

Images from Docker Hub and Quay.io have properties, such as the repository path
and the registry host, that you can use to choose which profiles apply to them.
See
[Artifact properties set by the DockerHub and Quay providers](profile_selectors.md#artifact-properties-set-by-the-dockerhub-and-quay-providers).
