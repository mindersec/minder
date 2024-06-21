---
title: GitHub
sidebar_position: 10
---

# Providers

A provider connects Minder to your software supply chain. It lets Minder know where to look for your repositories, artifacts,
and other entities are, in order to make them available for registration. It also tells Minder how to interact with your
supply chain to enable features such as alerting and remediation. Finally, it handles the way Minder authenticates
to the external service.

The currently supported providers are:
* GitHub

Stay tuned as we add more providers in the future!

## Enrolling a provider

To enroll GitHub as a provider, use the following command:
```
minder provider enroll
```

Once a provider is enrolled, public repositories from that provider can be registered with Minder. Security profiles
can then be applied to the registered repositories, giving you an overview of your security posture and providing
remediations to improve your security posture.

## Enrolling a provider with configuration

To specify provider configuration on enrollment, add the `--provider-config` flag and specify the path to the provider configuration file. For example:
```bash
minder provider enroll --provider-config /path/to/github-app-config.json
```

The provider configuration file should be a JSON file with the following format:
```json
{
  "github_app": {},
  "auto_registration": {
    "entities": {
      "repository": {
        "enabled": true
      }
    }
  }
}
```

See the following section for provider configuration reference

### GitHub App Provider Configuration reference

The GitHub App provider has the following configuration options:

* `auto_registration` (object): Configuration for the provider auto-registration feature
  * `entities` (object): Configuration for auto-registering different entities
    * `repository` (object): Configuration for auto-registering repositories
      * `enabled` (boolean): Whether to auto-register repositories. Default is `false`.