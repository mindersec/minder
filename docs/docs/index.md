---
title: Minder
sidebar_position: 1
---

![minder logo](./images/Minder_darkMode.png)

# What is Minder?

Minder by Stacklok is an open source platform that helps development teams and open source communities build more secure software, and prove to others that what they’ve built is secure. Minder helps project owners proactively manage their security posture by providing a set of checks and policies to minimize risk along the software supply chain, and attest their security practices to downstream consumers. 

Minder allows users to enroll repositories and define policy to ensure repositories and artifacts are configured consistently and securely. Policies can be set to alert only or autoremediate. Minder provides a predefined set of rules and can also be configured to apply custom rules.

Minder can be deployed as a Helm chart and provides a CLI tool ‘minder’. Minder is designed to be extensible, allowing users to integrate with their existing tooling and processes. 

## Features

* **Repo configuration and security:** Simplify configuration and management of security settings and policies across repos.
* **Proactive security enforcement:** Continuously enforce best practice security configurations by setting granular policies to alert only or auto-remediate.
* **Artifact attestation:** Continuously verify that packages are signed to ensure they’re tamper-proof, using the open source project Sigstore.
* **Dependency management:** Manage dependency security posture by helping developers make better choices and enforcing controls. Minder is integrated with [Trusty by Stacklok](http://trustypkg.dev) to enable policy-driven dependency management based on the risk level of dependencies.

## Minder Cloud

Stacklok, the company behind Minder, provides a free-to-use SaaS version of Minder that includes a UI (for public repositories only).
You can access Minder Cloud at [https://cloud.stacklok.com/](https://cloud.stacklok.com) and access documentation at [https://docs.stacklok.com/minder](https://docs.stacklok.com/minder).

## Status

Minder is currently in _Alpha_, meaning that it is not ready for production use: features and functionality is likely to change.

The public roadmap for Minder is available here: [link](./about/roadmap.md)
