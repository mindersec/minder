---
title: Dependabot
sidebar_position: 30
---

# Dependabot Rule

The following rule type is available for Dependabot.

## `dependabot_configured` - Verifies that Dependabot is configured for the repository

This rule allows you to monitor if Dependabot is enabled for automated dependency updates for repositories.
It is recommended that repositories have some form of automated dependency updates enabled
to ensure that vulnerabilities are not introduced into the codebase.

### Entity
- `repository`

### Type
- `dependabot_configured`

### Rule parameters
- None

### Rule definition options

The `dependabot_configured` rule supports the following options:
- `package_ecosystem (string)` - The package ecosystem to check for updates
    - The package ecosystem that the rule applies to. For example, `gomod`, `npm`, `docker`, `github-actions`, etc.
- `schedule_interval (string)` - The interval at which to check for updates
    - The interval that the rule should be evaluated. For example, `daily`, `weekly`, `monthly`, etc.
- `apply_if_file (string)` - Optional. The file to check for to determine if the rule should be applied 
    - If specified, the rule will only be evaluated if the given file exists. This is useful for rules that are only applicable to certain types of repositories.