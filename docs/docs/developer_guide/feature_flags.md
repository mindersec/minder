---
title: Feature flags
sidebar_position: 20
---

# Using Feature Flags

Minder is using [OpenFeature](https://openfeature.dev/) for feature flags.  For more complex configuration, refer to that documentation.  With that said, our goals are to allow for _simple, straightforward_ usage of feature flags to **allow merging code which is complete before the entire feature is complete**.

## When to use feature flags

Appropriate usages of feature flags:

* **Stage Changes**.  Use a feature flag to (for example) add the ability to write values in one PR, and the ability to operate on them in another PR.  By putting all the functionality behind a feature flag, it can be released all at once (when the documentation is complete).  Depending on the functionality, this may also be used to achieve a **staged rollout** across a larger population of users, starting with people willing to beta-test the feature.

* **Kill Switch**.  For features which introduce new load (e.g. 10x GitHub API token usage) or new access patterns (e.g. change message durability), feature flags can provide a quick way to be able to enable or revert changes without needing to build and push a new binary or config option (particularly if other code has changed in the meantime).  In this case, feature flags provide a consistent way of managing configuration as an alternative to `internal/config/server`.  Note that _feature flags_ affect a particular invocation (based on the user or project in question), while _config_ generally affects all behavior of the server.

* **Feature acceptance testing** (A/B testing).  When running Minder as a service, the Stacklok team may want to perform large-scale evaluation of whether a feature is useful to end-users.  Feature flags can allow comparing the usage of two groups with and without the feature enabled.

### Inappropriate Use Of Feature Flags

We expect that feature flags will generally be short-lived (a few months in most cases).  There are costs (testing, maintenance, complexity, and general opportunity costs) to maintaing two code paths, so we aim to retire feature flags once the feature is considered "stable".  Here are some examples of alternative mechanisms to use for long-term behavior changes:

* **Server Configuration**.  See [`internal/config/server`](https://github.com/stacklok/minder/tree/main/internal/config/server) for long-term options that should be on or off at server startup and don't need to change based on the invocation.

* **Entitlements**.  See [`internal/projects/features`](https://github.com/stacklok/minder/tree/main/internal/projects/features) for functionality that should be able to be turned on or off on a per-project basis (for example, for paid customers).

## How to Use Feature Flags

If you're working on a new Minder feature and want to merge it incrementally, check out [this code (linked to commit)](https://github.com/stacklok/minder/blob/d8f7d5709540bd33a2200adc2dbd330bbeceae86/internal/controlplane/handlers_authz.go#L222) for an example.  The process is basically:

1. Add a feature flag declaration to [`internal/flags/constants.go`](https://github.com/stacklok/minder/blob/main/internal/flags/constants.go)

1. At the call site(s), put the new functionality behind `if flags.Bool(ctx, s.featureFlags, flags.MyFlagName) {...`

1. You can use the [`flags.FakeClient`](https://github.com/stacklok/minder/blob/main/internal/flags/test_client.go) in tests to test the new code path as well as the old one.

Using `flags.Bool` from our own repo will enable a couple bits of default behavior over OpenFeature:

* We enforce that the default value of the flag is "off", so you can't end up with the confusing `disable_feature=false` in a config.
* We extract the user, project, and provider from `ctx`, so you don't need to.
* Eventually, we'll also record the flag settings in our telemetry records (WIP)

## Using Flags During Development

You can create a `flags-config.yaml` in the root Minder directory when running with `make run-docker`, and the file (and future changes) will be mapped into the Minder container, so you can make changes live.  The `flags-config.yaml` uses the [GoFeatureFlag format](https://gofeatureflag.org/docs/configure_flag/flag_format), and is in the repo's `.gitignore`, so you don't need to worry about accidentally checking it in.  Note that the Minder server currently rechecks the flag configuration once a minute, so it may take a minute or two for flags changes to be visible.

When deploying as a Helm chart, you can create a ConfigMap named `minder-flags` containing a key `flags-config.yaml`, and it will be mounted into the container.  Again, changes to the `minder-flags` ConfigMap will be updated in the Minder server within about 2 minutes of update.
