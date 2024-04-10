---
title: Roadmap
sidebar_position: 70
---

# Roadmap
## About this roadmap

This roadmap should serve as a reference point for Minder users and community members to understand where the project is heading. The roadmap is where you can learn about what features we're working on, what stage they're in, and when we expect to bring them to you. Priorities and requirements may change based on community feedback, roadblocks encountered, community contributions, and other factors. If you depend on a specific item, we encourage you to reach out to Stacklok to get updated status information, or help us deliver that feature by contributing to Minder.

## How to contribute

Have any questions or comments about items on the Minder roadmap? Share your feedback via [Minder GitHub Discussions](https://github.com/stacklok/minder/discussions). 


_Last updated: November 2023_

## In progress
* **Report CVEs, Trusty scores, and license info for dependencies in connected repos (with drift detection):** Identify dependencies in connected GitHub repositories and show CVEs, Trusty scores, and license information including any changes over time.
* **Additional policy capabilities to improve user experience:** Add the ability to edit/update policies, and provide a policy violation event stream that provides additional detail beyond the latest status.
* **Manage access policies with built-in roles:** Assign users a built-in role (e.g., admin, edit, view) on a resource managed in Minder.
* **Create Project(s) and add repos (domain model):** Group multiple GitHub repositories into a Project to simplify policy management.
* **Register an entire org to automatically add new repos:** Register an entire GitHub organization instead of a single repo; any newly created repos will automatically be added to Minder to simplify policy management.
* **Automate the signing of packages to ensure they are tamper-proof:** Use Sigstore to sign packages and containers based on policy.

## Next
* **Report CVEs, Trusty scores, and license info for ingested SBOMs:** Ingest SBOMS and identify dependencies; show CVEs, Trusty scores, and license information including any changes over time.
* **Block PRs based on Trusty scores:** In addition to adding comments to pull requests (as is currently available), add the option to block pull requests as a policy remediation.
* **Create policy to manage licenses in PRs:** Add a rule type to block and/or add comments to pull requests based on the licenses of the dependencies they import.
* **Automate the generation and signing of SLSA provenance statements:** Enable users to generate SLSA provenance statements (e.g. through SLSA GitHub generator) and sign them with Sigstore.
* **Export a Minder 'badge/certification' that shows what practices a project followed:** Create a badge that OSS maintainers and enterprise developers can create and share with others that asserts the Minder practices and policies their projects follow.

## Future considerations
* **Enroll GitLab and Bitbucket repositories:** In addition to managing GitHub repositories, enable users to manage configuration and policy for other providers.
* **Temporary permissions to providers vs. long-running:** Policy remediation currently requires long-running permissions to providers such as GitHub; provide the option to enable temporary permissions.
* **Create nested hierarchy of Projects:** Enable users to create multiple levels of Projects, where policies are inherited through levels of the tree, to simplify policy management.
* **Move a resource or Project between Projects:** Enable users to move resources from one Project to another, and update policies accordingly.
* **Create PRs for dependency updates:** As a policy autoremediation option, enable Minder to automatically create pull requests to update dependencies based on vulnerabilities, Trusty scores, or license changes.
* **Drive policy through git (config management):** Enable users to dynamically create and maintain policies from other sources, e.g. Git, allowing for easier policy maintenance and the ability to manage policies through GitOps workflows.
* **Ensure a project has a license:** A check that determines if a project has published a license.
* **Perform check for basic repo config:** A check that determines if a repository has basic user-specified configuration applied, e.g. public/private, default branch name.
* **Run package behavior analysis tool:** Enable a policy to continually run the [OSSF Package Analysis tool](https://github.com/ossf/package-analysis), which analyzes the capabilities of packages available on open source repositories. The project looks for behaviors that indicate malicious software and  tracks changes in how packages behave over time, to identify when previously safe software begins acting suspiciously.
* **Help package authors improve Activity score in Trusty:** Provide guidance and/or policy to improve key Trusty Activity score features (e.g., open issues, active contributors).
* **Help package authors improve Risk Flags score in Trusty:** Provide guidance and/or policy to improve key Trusty Risk Flags score features (e.g., package description, versions).
* **Enable secrets scanning and code scanning with additional open source and commercial tools:** Provide integrations to run scanning tools automatically from Minder (e.g. Synk, Trivy).
* **Generate SBOMs:** Enable users to automatically create and sign SBOMs.
