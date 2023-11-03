---
title: Profile Introduction
sidebar_position: 10
---

# Profile Introduction

Mediator allows you to define profiles for your software supply chain.

The anatomy of a profile is the profile itself, which outlines the rules to be
checked, the rule types, and the evaluation engine.

As of time of writing, mediator supports the following evaluation engines:

* **[Open Profile Agent](https://www.openprofileagent.org/)** (OPA) profile language.
* **[JQ](https://jqlang.github.io/jq/)** - a lightweight and flexible command-line JSON processor.

Each engine is designed to be extensible, allowing you to integrate your own
logic and processes.

Please see the [examples](https://github.com/stacklok/minder/tree/main/examples) directory for examples of profiles, and [Manage Profiles](./manage_profiles.md) for more details on how to set up profiles and rule types.
