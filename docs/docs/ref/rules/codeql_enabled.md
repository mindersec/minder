---
title: Code scanning
sidebar_position: 40
---

# Code Scanning (CodeQL) Rule

The following rule type is available for Code Scanning (CodeQL).

## `codeql_enabled` - Verifies that CodeQL is enabled for the repository

This rule allows you to monitor if code scanning via CodeQL is enabled for your repositories. 
CodeQL is a tool that can be used to analyze code for security vulnerabilities.
It is recommended that repositories have some form of static analysis enabled
to ensure that vulnerabilities are not introduced into the codebase.

### Entity
- `repository`

### Type
- `codeql_enabled`

### Rule parameters
- None

### Rule definition options

The `codeql_enabled` rule supports the following options:
- `languages ([]string)` - Only applicable for remediation. Sets the CodeQL languages to use in the workflow.
    - CodeQL supports `c-cpp`, `csharp`, `go`, `java-kotlin`, `javascript-typescript`, `python`, `ruby`, `swift`
- `schedule_interval (string, cron format)` - Only applicable for remediation. Sets the schedule interval for the workflow.
    - Example: `20 14 * * 1` (every Monday at 2:20pm)
