---
title: Using Minder with GitHub Advanced Security
sidebar_position: 90
---

## Prerequisites

* The `minder` CLI application
* A Stacklok account
* At least one GitHub public repository (an Advanced Security license is not required to apply Advanced Security settings to public repos)

## Overview

[GitHub Advanced Security](https://docs.github.com/en/get-started/learning-about-github/about-github-advanced-security) includes a set of security features that can be enabled for free on public repos, and with the purchase of an enterprise license for private repos. You can use Minder to automatically enable and enforce Advanced Security features across a group of repositories, to avoid manual enablement and monitoring. 

Note: Minder is currently only available for public repos to support open source projects. Please contact us if you’re interested in using Minder with GitHub Advanced Security for your private repos. 

## GitHub Advanced Security Features

* **[Code scanning](https://docs.github.com/en/code-security/code-scanning/introduction-to-code-scanning/about-code-scanning)**: Analyzes code in your repo for security vulnerabilities and coding errors. This practice helps find vulnerabilities or errors in your code before you commit, to help you avoid rework and issues down the road. 
* **[Secret scanning](https://docs.github.com/en/code-security/secret-scanning/about-secret-scanning)**: Scans your repos and notifies you about any leaked secrets, like tokens and private keys. Secret leakage can happen particularly when a formerly private repo moves to public, or joins your project’s group of repos. Having this type of scanning automatically enabled and enforced for all repos in your project can help prevent unintentional and unknown leakage. 
* **[Dependabot alerts and dependency review](https://docs.github.com/en/code-security/supply-chain-security/understanding-your-software-supply-chain/about-dependency-review)**: GitHub enables dependency graphs for public repos by default. The dependency graph helps you visualize the dependencies in your repo; which projects are using them; and vulnerability data for those dependencies. When Dependabot alerts are enabled for your repo, along with the dependency graph, you can get an alert when a new security advisory is detected for those dependencies, or when the dependency graph changes (like through a version bump or a code change to the dependency itself). 

These features can be quickly applied and continuously enforced across multiple repos using Minder’s [existing resource rule types](https://github.com/stacklok/minder-rules-and-profiles) and our [GitHub Advanced Security profile](https://github.com/stacklok/minder-rules-and-profiles/blob/main/profiles/github/ghas.yaml) as a reference. 

## Applying and enforcing GitHub Advanced Security features using Minder

**Step 1: Enroll the GitHub provider.** \
This allows Minder to manage your GitHub repositories. The following command will prompt you to log in to your GitHub account:
```bash
minder provider enroll
```

**Step 2: Register your GitHub repos.** \
Choose which repos for which you want to apply and enforce GitHub Advanced security settings. You can register a set of repositories by name, using the command: 
```bash
minder repo register --name "owner/repo1,owner/repo2"
```

**Step 3: Create rule types for GitHub Advanced Security settings using Minder’s existing resource rule types.** \
To easily add rules for GitHub Advanced Security settings to a profile and apply those rules to your repos, you can use Minder’s existing reference rules. 

First, you can fetch all of our reference rules by cloning the [minder-rules-and-profiles](https://github.com/stacklok/minder-rules-and-profiles) repository:
```bash
git clone https://github.com/stacklok/minder-rules-and-profiles.git
```

In that directory, you will find all of our reference rules and profiles: 
```bash
cd minder-rules-and-profiles
```

You can now choose from the reference rules in this list to apply to your profile. Our [GitHub Advanced Security profile](https://github.com/stacklok/minder-rules-and-profiles/blob/main/profiles/github/ghas.yaml) provides an example of a profile with GitHub Advanced Security rule types added. 

You can also pick and choose which rule types to create, based on which GitHub security settings you want to enable. The below reference rule types will enable GitHub Advanced Security settings: 

* codeql_enabled.yaml
* secret_scanning.yaml
* secret_push_protection.yaml
* dependabot_configured.yaml

To create a rule type, you can use the following command (using secret scanning as an example):
```bash
minder ruletype create -f rule-types/github/secret_scanning.yaml
```

Minder also includes reference rule types for other security features for GitHub, like branch protections and creating allowlists for GitHub Actions. To enable those additional security features, you can also choose to create rule types for all available GitHub security features by using this command:
```bash
minder ruletype create -f rule-types/github/
```

**Step 4: Create a profile to apply GitHub Advanced Security Settings to your repos.** \
After you’ve created your rule types, you can set up a profile that checks to make sure that these settings are in place for your registered repos. Profiles represent a configuration that you can apply to a group of repos. 
Start by creating a file named profile.yaml, and add some basic information:
```
version: v1 
type: profile 
name: my-ghas-profile 
context: provider: github
```

Register the rules that you just created:
```
repository: 
 - type: secret_scanning 
def: 
  enabled: true
```

Choose to enable alerting and autoremediation. With autoremediation enabled, Minder will continuously check to ensure that these settings are applied to your registered repos, and take action when the settings are not applied (for example, by opening a PR with a fix, or re-applying the setting). 
```
alert: "on"
remediate: "on"
```

And then create your profile in Minder:
```bash
minder profile create -f profile.yaml
```

Check the status of your profile and see which repositories satisfy the rules by running:
```bash
minder profile status list --name my-ghas-profile --detailed
```

With these steps in place, Minder will continuously apply and enforce GitHub Advanced Security settings to the repos you’ve registered for your profile. 
