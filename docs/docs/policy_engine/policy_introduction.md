---
id: policy_introduction
title: Policy Introduction
sidebar_position: 1
slug: /policy_introduction
displayed_sidebar: mediator
---

# Policy Introduction

Mediator allows you to define policies for your software supply chain.

The anatomy of a policy is the policy itself, which outlines the rules to be
checked, the rule types, and the evaluation engine.

As of time of writing, mediator supports the following evaluation engines:

* **[Open Policy Agent](https://www.openpolicyagent.org/)** (OPA) policy language.
* **[JQ](https://jqlang.github.io/jq/)** - a lightweight and flexible command-line JSON processor.

Each engine is designed to be extensible, allowing you to integrate your own
logic and processes.

Please see the [examples](https://github.com/stacklok/mediator/tree/main/examples) directory for examples of policies, and [Manage Policies](./manage_policies.md) for more details on how to set up policies and rule types.
