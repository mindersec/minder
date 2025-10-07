### What Is a *data source*?

A *data source* in Minder is a component that enriches your existing entities with additional information. (For broader context, see [Data Sources in Minder](https://mindersec.github.io/understand/key_concepts#data-sources).) 

While providers in Minder typically create or manage entities (e.g., repositories, issues, PRs, or services) and handle their lifecycle (registration, webhooks, etc.), *data sources* contribute extra context or data points about those already-created entities.

- **Providers**: Create and manage entities (e.g., ingest a GitHub repository).  
- **Data sources**: Enrich or augment existing entities with external or supplementary data (e.g., fetch known vulnerabilities for a package).

#### Key Attributes

- They do **not** create entities. Data sources only enhance an entity already known to Minder.  
- They can reference external services—for instance, pulling in vulnerability data from OSV or ClearlyDefined or a malware scanning service.  
- They have arguments that help shape the queries or requests the data source makes against external systems (e.g., specifying the package name, ecosystem, or version).
- They can leverage the authentication from the current Provider to fetch additional authenticated data after the initial ingestion.

---

### Why Would You Use a *data source*?

You would create a data source in Minder whenever you need additional information about an entity that was not included in the initial ingest. Common scenarios include:

- **Followup queries**: In some cases, it may be necessary to fetch additional information to evaluate the state of the entity based on data from the initial ingestion.  (For example, checking whether a workflow action has been passing after determining the relevant action.)
- **Enriching dependencies**: If a provider ingests a list of dependencies from a repository, a data source can query a vulnerability database (like OSV or ClearlyDefined) to see if any are known to be risky *from a security or licensing point of view*.
- **Performing security checks**: A data source might call out to a malware scanner or an external REST service to verify the integrity of binaries or tarballs.
- **Fetching attestation data**: If you need statements of provenance or supply-chain attestations from a separate system, a data source can gather this data for your entity.
- **Aggregating metadata from multiple sources**: For instance, combining ClearlyDefined’s scoring data with an internal database that tracks maintainers, deprecation status, or license data.

Essentially, data sources let Minder orchestrate external queries that feed into policy evaluations (e.g., Rego constraints) to create richer compliance, security, or operational checks.

---

### Defining a *data source*

When you invoke a data source in a Rego policy, you typically provide a set of arguments. These arguments tell the data source *what* to fetch or *how* to fetch it.

For example, consider the two YAML snippets below:

```yaml
version: v1
type: data-source
name: ghapi
context: {}
rest:
  providerAuth: true
  def:
    license:
      endpoint: https://api.github.com/repos/{owner}/{repo}/license
      parse: json
      input_schema:
        type: object
        properties:
          owner:
            type: string
          repo:
            type: string
    repo_config:
      endpoint: https://api.github.com/repos/{owner}/{repo}
      parse: json
      input_schema:
        type: object
        properties:
          owner:
            type: string
          repo:
            type: string
    private_vuln_reporting:
      endpoint: https://api.github.com/repos/{owner}/{repo}/private-vulnerability-reporting
      parse: json
      input_schema:
        type: object
        properties:
          owner:
            type: string
          repo:
            type: string
    graphql:
      endpoint: https://api.github.com/graphql
      method: POST
      body_from_field: query
      input_schema:
        query:
          type: object
          properties:
            query:
              type: object
          # We don't specify properties here, but a caller might use:
          # {concat("", "repository(name:\"", repo "\", owner:\"", owner "\"") {rulesets(first:20) ...}}
      fallback:
        http_status: 200
        body: '{results: [], error: "Error fetching data"}'
```

#### Key Fields

- **version / type / name**: Defines this resource as a data source called `ghapi`.
- **context**: Typically holds the project context. Here it’s `{}`, meaning it’s globally available (or within your chosen project scope).
- **rest**: Declares REST-based operations. If `providerAuth` is set to `true`, the provider's authentication mechanism will be used if the method's endpoint matches the provider's URL. Under `def`, we define three endpoints:
  - `license` → Fetches repository license info from GitHub
  - `repo_config` → Fetches general repo config (e.g., visibility, description, forks, watchers)
  - `private_vuln_reporting` → Fetches whether the repository has private vulnerability reporting enabled
  - `graphql` → Performs a GraphQL query

Each method defined in the rest endpoints has the following fields:

- **endpoint**: A [RFC 6570](https://tools.ietf.org/html/rfc6570) template URI with the supplied arguments (see [Using a data source in a Rule](#using-a-data-source-in-a-rule)).
- **method**: The HTTP method to invoke.  Defaults to `GET`.
- **headers**: A key-value map of static headers to add to the request.
- **bodyobj**: Specifies the request body as a static JSON object.
- **bodystr**: Specifies the request body as a static string.
- **body_from_field**: Specifies that the request body should be produced from the specified argument. Objects will be converted to JSON representation, while strings will be used as an exact request body.
- **parse**: Indicates the response format (`json`). If unset, the result will be the body as a string.
- **input_schema**: Uses JSON Schema to define the parameters needed by this data source in Rego. If you specify `input_schema` incorrectly, you will receive an error at runtime, helping ensure that the data you pass in matches what the data source expects.
  - *(Note: You can define additional properties as needed, but only fields explicitly handled by the data source code will be recognized.)*
- **expected_status**: Defines the expected response code. The default expected code is 200. If an unexpected response code is received, an error will be raised.
- **fallback**: If the request fails after 4 attempts and a fallback is defined, the specified **http_status** and **body** will be returned.

---

### Using a *data source* in a Rule

Below is the definition of a rule type named **osps-vm-05**. Notice how it includes `ghapi` under `data_sources` and calls `ghapi.private_vuln_reporting` in the Rego policy. We need the data source here because the rule checks private vulnerability reporting—information that isn’t part of the standard entity data and must be fetched from GitHub’s API.

```yaml
---
version: v1
release_phase: alpha
type: rule-type
name: osps-vm-05
display_name: Contacts and process for reporting vulnerabilities is published
short_failure_message: No contacts or process for reporting vulnerabilities was found
severity:
  value: info
context:
  provider: github
description: |
  This rule ensures that the repository provides a clear process and contact information 
  for reporting vulnerabilities. 
  It checks for the presence of a SECURITY.md file containing relevant reporting 
  details or verifies if GitHub's private vulnerability reporting feature is enabled.
guidance: |
  To address this issue:
  1. Add a `SECURITY.md` file to your repository:
     - Ensure it includes instructions for reporting vulnerabilities, including contact details and a clear process.
     - Refer to [GitHub's documentation on SECURITY.md](https://docs.github.com/en/code-security).
  2. Alternatively, enable GitHub's private vulnerability reporting:
     - Navigate to the repository's "Settings" → "Security and Analysis."
     - Enable "Private vulnerability reporting."
def:
  in_entity: repository
  rule_schema: {}
  ingest:
    type: git
  eval:
    type: rego
    data_sources:
      - name: ghapi
    rego:
      type: deny-by-default
      def: |
        package minder

        import rego.v1
        default allow := false

        ######################################################################
        # ALLOW if there's a SECURITY.md file with the word "report"
        ######################################################################
        allow if {
          # List any file starting with SECURITY (e.g. SECURITY.md, SECURITY.txt)
          files := file.ls_glob("./SECURITY*")
          count(files) > 0

          # Read the first matching file's content
          content := lower(file.read(files[0]))

          # Check if "report" is in the file content
          contains(content, "report")
        }

        ######################################################################
        # ALLOW if GitHub private vulnerability reporting is enabled
        ######################################################################
        allow if {
          # Query the GitHub API to check private vulnerability reporting
          out = minder.datasource.ghapi.private_vuln_reporting({
            "owner": input.properties["github/repo_owner"],
            "repo":  input.properties["github/repo_name"]
          })

          # out.body.enabled == true if private vulnerability reporting is turned on
          out.body.enabled == true
        }
```

#### Rule Evaluation In Action

When the **osps-vm-05** rule is evaluated, Minder executes the following steps to determine whether there is a clear way for end-users to report vulnerabilities in the repository:

1. **Ingestion**  
   - `ingest: { type: git }` indicates this rule runs on a Git-based repository, allowing you to check for the existence of files like `SECURITY.md` by first copying the repository’s content.

2. **Data Sources**  
   - `data_sources: - name: ghapi` tells Minder to make the `ghapi` data source available within this rule.  
   - In your Rego code, you call the `private_vuln_reporting` endpoint by referencing `minder.datasource.ghapi.private_vuln_reporting(...)`.

3. **Rego Logic**  
   - The rule sets `default allow := false`, meaning the repository fails this check by default.  
   - The repository *passes* if *either* of these conditions is true:
     - **Condition A**: A `SECURITY.md` file (or similar) containing “report” is present.  
     - **Condition B**: GitHub’s “private vulnerability reporting” feature is enabled (via the GitHub API check).

4. **Violations**  
   - If neither condition is satisfied, the rule remains in a “deny” state, and Minder surfaces that as a violation, indicating the repository lacks a clear vulnerability reporting process.
