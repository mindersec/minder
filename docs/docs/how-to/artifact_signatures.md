---
title: Check artifact provenance
sidebar_position: 55
---

# Check Artifact Provenance

With Minder you can create rules that assert that artifacts built from your
repositories are built from trusted sources and using trusted workflows based
on their cryptographically signed provenance.

This is done by creating a profile which utilizes the `artifact_signature`
`rule_type`. 

## Prerequisites

* The `minder` CLI application
* A Stacklok account
* An enrolled Provider (e.g., GitHub)
* A repository that produces container images. At the moment Minder's artifact signature checks are only available for container images and only the `ghcr.io` registry is supported.

## Create the artifact provenance rule type

Fetch all the reference rules by cloning the [minder-rules-and-profiles repository](https://github.com/stacklok/minder-rules-and-profiles).

```
git clone https://github.com/stacklok/minder-rules-and-profiles.git
```

In that directory you can find all the reference rules and profiles.
```
cd minder-rules-and-profiles
```

Create the `artifact_signature` rule type in Minder:
```
minder ruletype create -f rule-types/github/artifact_signature.yaml
```

## Define a simple profile that checks artifact signatures

Next, create a profile that applies the rule type to the appropriate artifact.

The artifacts are referred to by name and tag. If the name is not specified,
the rule will match any artifact name. The tag can be specified either as a list
of tags using the `tags` parameter or as a regular expression using the `tag_regex`
parameter. If both are empty, the rule will match any tag. It is an error to specify
both `tags` and `tag_regex`.

Create a new file called `profile-artifact-simple.yaml`. The following example would match a container
image named `good-repo-go` with the `latest` tag. The profile would pass for any artifact that
has a signature, regardless of who signed it.

```yaml
---
# sample policy for validating artifact signatures
version: v1
type: profile
name: latest-artifact-simple
context:
  provider: github
artifact:
  - type: artifact_signature
    params:
      tags: [latest]
      name: good-repo-go
    def:
      is_signed: true
      is_verified: true
```

Create the profile in Minder:
```
minder profile create -f profile-artifact-simple.yaml
```

Once the profile is created, Minder will start checking the artifacts produced by the enrolled repositories
and the policy status will be updated accordingly. If the artifact is not matching the expected provenance
(for example someone pushes a new image to the registry without signing it), a
violation is presented via the profile status and an alert is raised.

## Define a more advanced profile that checks artifact provenance
As the next step, let's create a profile that checks the provenance of the artifact.
Create a new file called `profile-artifact-provenance.yaml`.

The profile would pass only if the container was
built from the `main` branch of the `good-repo-go` repository, using the `build-image-signed-ghat.yml`
workflow using a hosted github runner.

```yaml
---
# sample policy for validating artifact provenance
version: v1
type: profile
name: latest-artifact-hardened
context:
  provider: github
artifact:
  - type: artifact_signature
    params:
      tags: [latest]
      name: good-repo-go
    def:
      is_signed: true
      is_verified: true
      branch: main
      signer_identity: build-image-signed-ghat.yml
      runner_environment: github-hosted
      repository: https://github.com/mytestorg/good-repo-go
      cert_issuer: https://token.actions.githubusercontent.com
```

Create the profile in Minder:
```
minder profile create -f profile-artifact-provenance.yaml
```

Once the profile is created, Minder will start checking the artifacts produced
by the enrolled repositories and the policy status will be updated
accordingly. If the artifact is not matching the expected provenance (for
example someone pushes a new image to the registry after having signed the
image with their personal account or the image is built from a different
workflow or a different branch), a violation is presented via the profile
status and an alert is raised.
