---
title: Introduction to running Minder
sidebar_position: 5
---

# Introduction to running Minder

Minder is platform, comprising of a control plane, a CLI, a database and an identity provider.

The control plane runs two endpoints, a gRPC endpoint and a HTTP endpoint.

Minder is controlled and managed via the CLI application `minder`.

PostgreSQL is used as the database.

Keycloak is used as the identity provider.

Depending on your goal, there are two methods to get started running a Minder server
- If you are interested in contributing to Minder as a developer, you can build Minder from source, while deploying its dependencies via containers. Follow the [Installing a Development version](./run_the_server.md) instructions. 

- If you'd like to run Minder as an end user or a production application, you should install using Helm. This method requires you to provide and configure your own identity provider and database. Follow the [Installing a Production version](./installing_minder.md) instructions.
