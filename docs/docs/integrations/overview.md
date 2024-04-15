---
title: Overview
sidebar_position: 10
---

# Minder Integrations

Minder, as a platform, supports multiple integrations with different aspects of your supply chain,
as well as sources of information to make relevant decisions.

## Providers

Providers are integrations with external services that provide information about your supply chain.

Think of them as device drivers in an operating system. They provide a common interface to interact with different services.

For more information, see the [Provider Integrations](provider_integrations/github.md) documentation.

## Integration with other tools

Minder is aims to be vendor neutral. That is, it doesn't care nor prefer one tool over the other.
It's designed to be flexible and integrate with the tools you already use.

Examples of integrations include:

- Scanning tools (e.g., Trivy)
- CI/CD tools (e.g. GitHub Actions)
- Automated dependency update tools (e.g. Dependabot)

For more information, see the [OSS Integrations](community_integrations.md) documentation.

## Trusty

Trusty is a tool that helps you make better decisions about your dependencies. It provides a set
of heuristics to help you decide if a dependency is trustworthy or not. It's also developed by
your friends at Stacklok!

Trusty is integrated into Minder via a dedicated rule type.

For more information, see the [Trusty](trusty.md) documentation.
