---
title: Changelog
sidebar_position: 30
---

# Changelog

- **Profile selectors** - Sep 9, 2024  
   You can now specify which repositories a profile applies to using a Common
  Expression Language (CEL) grammar.

- **Rule evaluation history** - Sep 4, 2024  
   You can now see how your security rules have applied to your repositories,
  pull requests, and artifacts throughout time, in addition to their current
  state.

- **User management** - Aug 5, 2024  
   Minder organization administrators can now invite additional users to the
  organization, and can set users permissions.

- **Manage all GitHub repositories** - Jul 17, 2024  
   Minder can now (optionally) manage all repositories within a GitHub
  organization, including new repositories that are created. Administrators can
  continue to select individual repositories to manage.

- **Built-in rules** - Apr 6, 2024  
   Minder now includes all the rules in our
  [sample rules repository](https://github.com/mindersec/minder-rules-and-profiles/)
  in your new projects automatically. This means that you do not need to clone
  that repository or add those rule types to make use of them.

  To use them, prefix the rule name as it exists in the sample rules repository
  with `stacklok/`. For example:

  ```yaml
  ---
  version: v1
  type: profile
  name: uses-builtin-rules
  context:
    provider: github
  repository:
    - type: stacklok/secret_scanning
      def:
        enabled: true
  ```

  You can still define custom rules, or continue to use the rules that exist in
  the
  [sample rules repository](https://github.com/mindersec/minder-rules-and-profiles).

- **User roles** - Jan 30, 2024  
  You can now provide access control for users (eg: administrator, editor,
  viewer) in your project using
  [built-in roles](../user_management/user_roles.md).
