---
title: Minder
sidebar_position: 1
---

![mediator logo](./images/mediator.png)

# What is Minder?

Minder is an open platform to manage the security of your software supply chain.

Minder unifies the security of your software supply chain by providing a single
place to manage your security profiles and a central location to view and remediate
the results.

Minder is designed to be extensible, allowing you to integrate with your existing
tooling and processes.

## Features

- **GitHub integration** - Connects to GitHub to provide a single
  place to manage your repository security posture.
- **Profile management** - It enables developers to define security profiles for your
    software supply chain.
- **Attestation and Provenance** - Integrates with [sigstore](https://sigstore.dev/)
    [in-toto](https://in-toto.io/), [slsa](https://slsa.dev) and the
    [the-update-framework](https://theupdateframework.io/) to provide a way to verify the provenance of your software supply chain.

## Architecture

Minder consists of a single golang binary which requires a backing Postgres database.  For more details on the architecture, see the [System Architecture](./developer_guide/architecture) section.

## Status

Minder is currently in early development.
