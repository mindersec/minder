---
title: Registering repositories and creating profiles
sidebar_position: 10
---

## Goal
The goal of this tutorial is to register a GitHub repository, and create a profile that checks if secret scanning
is enabled on the registered repository.

## Prerequisites

* The `minder` CLI application
* A Stacklok account

## Enroll a provider
The first step is to tell Minder where to find your repositories.  
You do that by enrolling a provider.

In the example below, the chosen provider is GitHub, as indicated by the `--provider` flag.  
This will allow you to later enroll your account's repositories.

```
minder provider enroll --provider github
```

This command will open a window in your browser, prompting you to authorize Stacklok to access some data on GitHub.

## Register repositories
Once you have enrolled a provider, you can register repositories from that provider.

```
minder repo register --provider github
```
This command will show a list of the public repositories available for registration.

Navigate through the repositories using the arrow keys and select one or more repositories for registration 
by using the space key.  
Press the enter key once you have selected all the desired repositories.

You can see the list of repositories registered in Mediator.
```
minder repo list --provider github
```

## Creating and applying profiles
A profile is a set of rules that you apply to your registered repositories.
Before creating a profile, you need to ensure that all desired rule_types have been created in Minder.

Start by creating a rule that checks if secret scanning is enabled and creates a security advisory 
if secret scanning is not enabled.  
This is a reference rule provider by the Minder team.

Fetch all the reference rules by cloning the [minder-rules-and-profiles repository](https://github.com/stacklok/minder-rules-and-profiles).

```
git clone https://github.com/stacklok/minder-rules-and-profiles.git
```

In that directory you can find all the reference rules and profiles.
```
cd minder-rules-and-profiles
```

Create the `secret_scanning` rule type in Minder:
```
minder rule_type create -f rule-types/github/secret_scanning.yaml
```

Next, create a profile that applies the secret scanning rule.

Create a new file called `profile.yaml`.
Paste the following profile definition into the newly created file.

```yaml
---
version: v1
type: profile
name: github-profile
context:
  provider: github
alert: "on"
remediate: "off"
repository:
  - type: secret_scanning
    def:
      enabled: true
```

Create the profile in Minder:
```
minder profile create -f profile.yaml
```

Check the status of the profile:
```
./bin/minder profile_status list --profile github-profile
```
If all registered repositories have secret scanning enabled, you will see the `OVERALL STATUS` is `Success`, otherwise the 
overall status is `Failure`.

See a detailed view of which repositories satisfy the secret scanning rule:
```
./bin/minder profile_status list --profile github-profile --detailed
```

## Viewing alerts

Disable secret scanning in one of the registered repositories, by following these 
[instructions](https://docs.github.com/en/code-security/secret-scanning/configuring-secret-scanning-for-your-repositories)
provided by GitHub.

Navigate to the repository on GitHub, click on the Security tab and view the Security Advisories.  
Notice that there is a new advisory titled `mediator: profile github-profile failed with rule secret_scanning`.

Enable secret scanning in the same registered repository, by following these
[instructions](https://docs.github.com/en/code-security/secret-scanning/configuring-secret-scanning-for-your-repositories)
provided by GitHub.

Navigate to the repository on GitHub, click on the Security tab and view the Security Advisories.
Notice that the advisory titled `mediator: profile github-profile failed with rule secret_scanning` is now closed.
