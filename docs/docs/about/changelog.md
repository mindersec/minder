---
title: Changelog
sidebar_position: 30
---

# Changelog

This is the changelog for [Minder Cloud](https://cloud.stacklok.com/), the Minder service hosted by [Stacklok](https://stacklok.com/).

* **Minder Web** - Apr 16, 2024  
    Minder Cloud now has a graphical user interface available at [https://cloud.stacklok.com/](https://cloud.stacklok.com/).

* **Built-in rules** - Apr 6, 2024  
    Minder now includes all the rules in our [sample rules repository](https://github.com/stacklok/minder-rules-and-profiles/) in your new projects automatically. This means that you do not need to clone that repository or add those rule types to make use of them.

    To use them, prefix the rule name as it exists in the sample rules repository with `stacklok/`. For example:

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

    You can still define custom rules, or continue to use the rules that exist in the [sample rules repository](https://github.com/stacklok/minder-rules-and-profiles).

* **User roles** - Jan 30, 2024  
  You can now provide access control for users (eg: administrator, editor, viewer) in your project using [built-in roles](../user_management/user_roles.md).
