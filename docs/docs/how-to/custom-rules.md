# Writing Custom Rule Types

Minder's policy engine is flexible enough that you can write your own rule types to check for specific settings in your supply chain. This guide will walk you through the process of writing a custom rule type.

## Minder policies

Minder allows you to check and enforce that certain settings are set up for several stages in your supply chain. To configure those settings, you need to create a Profile. This profile is composed of several rules that represent the settings you want in your supply chain. These rules are actually instantiations of another handy object for Minder called Rule Types. These rule types define the nitty-gritty details of how the specific setting you care about will be checked, how you'll be alerted when something goes out of order, and how it will be automatically remediated.

You can browse a curated collection of rule types in the [rules and profiles repository])(https://github.com/stacklok/minder-rules-and-profiles).

Some of the rules include:

* Verifying if you have GitHub’s secret scanning enabled
* Verifying if your artifacts are signed and pushed to Sigstore
* Verifying that your branch protection settings are secure

## Rule Types

Rule types aren’t particularly complicated. They include the basic structure to get an observed state, evaluate the rule based on that observed state, do actions based on that state, and finally, give you some instructions in case you want to manage things manually.

The Rule Type object in YAML looks as follows:

```yaml
---
version: v1
type: rule-type
name: my_cool_new_rule
description: // Description goes here
guidance: // Guidance goes here
def:
  in_entity: repository  // what are we evaluating?
  param_schema: // parameters go here
  rule_schema: // rule definition schema goes here
  # Defines the configuration for ingesting data relevant for the rule
  ingest: // observed state gets fetched here
  eval: // evaluation goes here
  remediation: // fixing the issue goes here
  alert: // alerting goes here
```

The following are the components of a rule type:

* **Description**: What does the rule do? This is handy to browse through rule types when building a profile.
* **Guidance**: What do I do if this rule presents a “failure”? This is handy to inform folks of what to do in case they’re not using automated remediations.
* **in_entity**: What are we evaluating? This defines the entity that’s being evaluated. It could be repository, artifact, pull_request, and build_environment (coming soon).
* **param_schema**: Optional fields to pass to the ingestion (more on this later). This is handy if we need extra data to get the observed state of an entity.
* **rule_schema**: Optional fields to pass to the evaluation (more on this later). This is handy for customizing how a rule is evaluated.
* **Ingest**: This step defines how we get the observed state for an entity. It could be a simple REST call, a cloning of a git repo, or even a custom action if it’s a complex rule.
* **Eval**: This is the evaluation stage, which defines the actual rule evaluation.
* **Remediation**: How do we fix the issue? This defines the action to be taken when we need to fix an issue. This is what happens when you enable automated remediations.
* **Alert**: How do we notify folks about the issue? This may take the form of a GitHub Security Advisory, but we’ll support more alerting systems in the near future.

## Example: Automatically delete head branches

Let's write a rule type for checking that GitHub automatically deletes branches after a pull request has been merged. While this is not strictly a security setting, it is a good practice to keep your branches clean to avoid confusion.

### Ingestion / Evaluation

The first thing we need to figure out is how to get the observed state of what we want to evaluate on. This is the ingestion part.

Fortunately for us, GitHub keeps up-to-date and extensive documentation on their APIs. A quick internet search leads us to the relevant [Repositories API](https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#get-a-repository) where we can see that a call to the `/repos/OWNER/REPO` endpoint gives us the following key: `delete_branch_on_merge`.

So, by now we know that we may fetch this information via a simple REST call.

The ingestion piece would then look as follows:

```yaml
---
def:
  ...
  ingest:
    type: rest
    rest:
      endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}"
      parse: json
```

While you could hard-code the user/org and name of the repository you want to evaluate, that kind of rule is not handy, especially if you want to enroll multiple repositories in Minder. Thus, Minder has a templating system that allows you to base multiple parts of the rule type on the entity you’re evaluating (remember the in_entity part of the rule type?). The fields you may use are part of the entity’s protobuf, which can be found in [our documentation](https://minder-docs.stacklok.dev/ref/proto#repository).

Now, we want to tell Minder what to actually evaluate from that state. This is the evaluation step. In our case, we want to verify that delete_branch_on_merge is set to true. For our intent, we have a very simple evaluation driver that will do the trick just fine! That is the [jq evaluation type](https://jqlang.github.io/jq/).

I understand this is not a setting that everybody would want, and, in fact, some folks might want that setting to be off. This is something we can achieve with a simple toggle. To do it, we need to add a rule_schema to our rule, which would allow us to have a configurable setting in our rule.

The evaluation would look as follows:

```yaml
---
def:
  rule_schema:
    type: object
    properties:
      enabled:
        type: boolean
    required:
      - enabled
  eval:
    type: jq
    jq:
    - ingested:
        def: '.delete_branch_on_merge'
      profile:
        def: ".enabled"
```



The rule type above now allows us to compare the delete_branch_on_merge setting we got from the GitHub call, and evaluate it against the enabled setting we've registered for our rule type.

### Alerting

We'll now describe how you may get a notification if your repository doesn’t adhere to the rule. This is as simple as adding the following to the manifest:

```yaml
---
def:
  alert:
    type: security_advisory
    security_advisory:
      severity: "low"
```

This will create a security advisory in your GitHub repository that you’ll be able to browse for information. Minder knows already what information to fill-in to make the alert relevant.

### Remediation

Minder has the ability to auto-fix issues that it finds in your supply chain, let’s add an automated fix to this rule! Similarly to ingestion, remediations also have several flavors or types. For our case, a simple REST remediation suffices.

Let’s see how it would look:

```yaml
---
def:
  remediate:
    type: rest
    rest:
      method: PATCH
      endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}"
      body: |
        { "delete_branch_on_merge": {{ .Profile.enabled }} }
```

This effectively would do a PATCH REST call to the GitHub API if it finds that the rule is out of compliance. We’re able to parametrize the call with whatever we defined in the profile using golang templates (that’s the `{{ .Profile.enabled }}` section you see in the message’s body).

### Description & Guidance

There are a couple of sections that allow us to give information to rule type users about the rule and what to do with it. These are the description and guidance. The description is simply a textual representation of what the rule type should do. Guidance is the text that will show up if the rule fails. Guidance is relevant when automatic remediations are not enabled, and we want to give folks instructions on what to do to fix the issue.

For our rule, they will look as follows:

```yaml
---
version: v1
type: rule-type
name: my_cool_new_rule
context:
  provider: github
description: |
  This rule verifies that branches are deleted automatically once a pull
  request merges.
guidance: |
  To manage whether branches should be automatically deleted for your repository
  you need to toggle the "Automatically delete head branches" setting in the
  general configuration of your repository.

  For more information, see the GitHub documentation on the topic: https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-the-automatic-deletion-of-branches
```

## Trying the rule out

The whole rule can be seen in the [Rules and Profiles GitHub repository](https://github.com/stacklok/minder-rules-and-profiles). In order to try it out, we’ll use the minder CLI, which points to the Minder server hosted by your friends at Stacklok.

Before continuing, make sure you use our Quickstart to install the CLI and enroll your GitHub repos.

Let’s create the rule:

```bash
$ minder ruletype create -f rules/github/automatic_branch_deletion.yaml                                   
```

Here, you can already see how the description gets displayed. This same description will be handy when browsing rules through `minder ruletype list`.

Let’s now try it out! We can call our rule in a profile as follows:

```yaml
---
version: v1
type: profile
name: degustation-profile
context:
  provider: github
alert: "on"
remediate: "off"
repository:
  - type: automatic_branch_deletion
    def:
      enabled: true
```

We’ll call this degustation-profile.yaml. Let’s create it!

```bash
$  minder profile create -f degustation-profile.yaml
```

Now, let's view the status of the profile:

```bash
$ minder profile status list -n degustation-profile -d
```

Depending on how your repository is set up, you may see a failure or a success. If you see a failure, you can enable automated remediations and see how Minder fixes the issue for you.

## Conclusion

We’ve now created a basic new rule for Minder. There are more ingestion types, rule evaluation engines, and remediation types that we can use today, and there will be more in the future! If you need support writing your own rule types, feel free to reach out to the Minder team.
