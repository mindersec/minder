---
title: Rule evaluations
sidebar_position: 50
---

# Rule evaluations

When Minder evaluates the [rules in your profiles](profiles.md), it records the state of those evaluations. When those rules are not satisfied because the criteria that you defined was not met, it will issue [alerts](alerts.md).

Minder evaluates the rules in your profile:
* When the repository is registered
* When the profile is updated
* When activity occurs within the repository

In a rule evaluation, you'll see:
* The time that a rule evaluation was performed
* The entity that was examined (a repository, artifact, or pull request)
* The rule that was evaluated
* The status of the evaluation
* Whether an [alert](alerts.md) was opened
* Whether [automatic remediation](remediations.md) was performed, and if so, its status

### Viewing rule evaluations

To view the rule evaluations, run [`minder history list`](../ref/cli/minder_history_list.md). You can query the history to only look at certain entities, profiles, or statuses.

### Evaluation status

The _status_ of a rule evaluation describes the outcome of executing the rule against an entity. Possible statuses are:

* **Success**: the entity was evaluated and is in compliance with the rule. For example, given the [`secret_scanning`](../ref/rules/secret_scanning) rule, this means that secret scanning is enabled on the repository being evaluated.
* **Failure**: the entity was evaluated and is _not_ in compliance with the rule. For example, given the [`secret_scanning`](../ref/rules/secret_scanning) rule, this means that secret scanning is _not_ enabled on the repository being evaluated.
* **Error**: the rule could not be evaluated for some reason. For example, the server being evaluated was not online or could not be contacted.
* **Skipped**: the rule is not configured for the entity. For example, given the [`secret_scanning`](../ref/rules/secret_scanning) rule, it can be configured to skip private repositories.

### Alert status

When a rule evaluation occurs, an [alert](alerts.md) may be created. Each rule evaluation has an alert status:

* **Success**: an alert was created
* **Failure**: there was an issue creating the alert; for example, GitHub failed to create a security advisory
* **Skipped**: the rule evaluation was successful, meaning an alert should not be created, or the profile is not configured to generate alerts

### Remediation status

* **Success**: the issue was automatically remediated
* **Failure**: the issue could not be automatically remediated
* **Skipped**: the rule evaluation was successful, meaning remediation should not be performed, or the profile is not configured to automatically remediate
