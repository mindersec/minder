"use strict";(self.webpackChunkminder_docs=self.webpackChunkminder_docs||[]).push([[806],{78632:(e,n,i)=>{i.r(n),i.d(n,{assets:()=>l,contentTitle:()=>a,default:()=>h,frontMatter:()=>o,metadata:()=>t,toc:()=>c});const t=JSON.parse('{"id":"how-to/writing-rules-in-rego","title":"Writing rules using Rego","description":"Minder\'s policy engine is able to use pluggable drivers for evaluating rules.","source":"@site/docs/how-to/writing-rules-in-rego.md","sourceDirName":"how-to","slug":"/how-to/writing-rules-in-rego","permalink":"/how-to/writing-rules-in-rego","draft":false,"unlisted":false,"tags":[],"version":"current","sidebarPosition":110,"frontMatter":{"title":"Writing rules using Rego","sidebar_position":110},"sidebar":"minder","previous":{"title":"Writing custom rule types","permalink":"/how-to/custom-rules"},"next":{"title":"Develop and debug rule types","permalink":"/how-to/mindev"}}');var s=i(74848),r=i(28453);const o={title:"Writing rules using Rego",sidebar_position:110},a=void 0,l={},c=[{value:"Writing rule types in Minder",id:"writing-rule-types-in-minder",level:2},{value:"Rego evaluation types",id:"rego-evaluation-types",level:2},{value:"Custom Rego functions",id:"custom-rego-functions",level:2},{value:"Example: CodeQL-enabled check",id:"example-codeql-enabled-check",level:2},{value:"Example: no &#39;latest&#39; tag in Dockerfile",id:"example-no-latest-tag-in-dockerfile",level:2},{value:"Example: security advisories check",id:"example-security-advisories-check",level:2},{value:"Linting",id:"linting",level:2}];function d(e){const n={a:"a",code:"code",em:"em",h2:"h2",li:"li",p:"p",pre:"pre",strong:"strong",ul:"ul",...(0,r.R)(),...e.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(n.p,{children:"Minder's policy engine is able to use pluggable drivers for evaluating rules.\nRego is a language specifically designed for expressing policies in a clear and\nconcise manner. Its declarative syntax makes it an excellent choice for defining\npolicy logic. In the context of Minder, Rego plays a central role in crafting\nrule types, which are used to enforce security policies."}),"\n",(0,s.jsx)(n.h2,{id:"writing-rule-types-in-minder",children:"Writing rule types in Minder"}),"\n",(0,s.jsx)(n.p,{children:"Minder organizes policies into rule types, each with specific sections defining\nhow policies are ingested, evaluated, and acted upon. Rule types are then called\nwithin profiles to express the security posture of your organization. Let's\ndelve into the essential components of a Minder rule type:"}),"\n",(0,s.jsxs)(n.ul,{children:["\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsx)(n.p,{children:"Ingesting data: Fetching relevant data, often from external sources like\nGitHub API."}),"\n"]}),"\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsx)(n.p,{children:"Evaluation: Applying policy logic to the ingested data. Minder offers a set of\nengines to evaluate data: jq and rego being general-purpose engines, while\nStacklok Insight and vulncheck are more use case-specific ones."}),"\n"]}),"\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsx)(n.p,{children:"Remediation and alerting: Taking actions or providing notifications based on\nevaluation results. E.g. creating a pull request or generating a GitHub\nsecurity advisory."}),"\n"]}),"\n"]}),"\n",(0,s.jsx)(n.h2,{id:"rego-evaluation-types",children:"Rego evaluation types"}),"\n",(0,s.jsx)(n.p,{children:"With Rego being a flexible policy language, it allowed us to express policy\nchecks via different constructs. We chose to implement two in Minder:"}),"\n",(0,s.jsxs)(n.ul,{children:["\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsxs)(n.p,{children:[(0,s.jsx)(n.strong,{children:"deny-by-default"}),": Checks for an allowed boolean being set to true, and\ndenies the policy if it's not the case."]}),"\n"]}),"\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsxs)(n.p,{children:[(0,s.jsx)(n.strong,{children:"constraints"}),": Checks for violations in the given policy. This allows us to\nexpress the violations and output them in a user friendly-manner."]}),"\n"]}),"\n"]}),"\n",(0,s.jsx)(n.p,{children:"Note that these are known patterns in the OPA community, so we're not doing\nanything out of the ordinary here. Instead, we leverage best practices that have\nalready been established."}),"\n",(0,s.jsx)(n.h2,{id:"custom-rego-functions",children:"Custom Rego functions"}),"\n",(0,s.jsx)(n.p,{children:"Given the context in which Minder operates, we did need to add some custom\nfunctionality that OPA doesn't provide out of the box. Namely, we added the\nfollowing custom functions:"}),"\n",(0,s.jsxs)(n.ul,{children:["\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsxs)(n.p,{children:[(0,s.jsx)(n.strong,{children:"file.exists(filepath)"}),": Verifies that the given filepath exists in the Git\nrepository, returns a boolean."]}),"\n"]}),"\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsxs)(n.p,{children:[(0,s.jsx)(n.strong,{children:"file.read(filepath)"}),": Reads the contents of the given file in the Git\nrepository and returns the contents as a string."]}),"\n"]}),"\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsxs)(n.p,{children:[(0,s.jsx)(n.strong,{children:"file.ls(directory)"}),": Lists files in the given directory in the Git\nrepository, returning the filenames as an array of strings."]}),"\n"]}),"\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsxs)(n.p,{children:[(0,s.jsx)(n.strong,{children:"file.ls_glob(pattern)"}),": Lists files in the given directory in the Git\nrepository that match the given glob pattern, returning matched filenames as\nan array of strings."]}),"\n"]}),"\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsxs)(n.p,{children:[(0,s.jsx)(n.strong,{children:"file.http_type(filepath)"}),": Determines the HTTP (MIME) content type of the\ngiven file by\n",(0,s.jsx)(n.a,{href:"https://mimesniff.spec.whatwg.org/",children:"examining the first 512 bytes of the file"}),".\nIt returns the content type as a string."]}),"\n"]}),"\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsxs)(n.p,{children:[(0,s.jsx)(n.strong,{children:"file.walk(path)"}),": Walks the given path (directory or file) in the Git\nrepository and returns a list of paths to all regular files (not directories)\nas an array of strings."]}),"\n"]}),"\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsxs)(n.p,{children:[(0,s.jsx)(n.strong,{children:"github_workflow.ls_actions(directory)"}),": Lists all actions in the given\nGitHub workflow directory, returning the filenames as an array of strings."]}),"\n"]}),"\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsxs)(n.p,{children:[(0,s.jsx)(n.strong,{children:"parse_yaml"}),": Parses a YAML string into a JSON object. This implementation\nuses ",(0,s.jsx)(n.a,{href:"https://gopkg.in/yaml.v3",children:"https://gopkg.in/yaml.v3"}),", which avoids bugs when parsing ",(0,s.jsx)(n.code,{children:'"on"'})," as an\nobject ",(0,s.jsx)(n.em,{children:"key"})," (for example, in GitHub workflows)."]}),"\n"]}),"\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsxs)(n.p,{children:[(0,s.jsx)(n.strong,{children:"jq.is_true(object, query)"}),": Evaluates a jq query against the specified\nobject, returning ",(0,s.jsx)(n.code,{children:"true"})," if the query result is a true boolean value, andh\n",(0,s.jsx)(n.code,{children:"false"})," otherwise."]}),"\n"]}),"\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsxs)(n.p,{children:[(0,s.jsx)(n.strong,{children:"file.archive(paths)"}),": ",(0,s.jsx)(n.em,{children:"(experimental)"})," Builds a ",(0,s.jsx)(n.code,{children:".tar.gz"})," format archive\ncontaining all files under the given paths. Returns the archive contents as a\n(binary) string."]}),"\n"]}),"\n"]}),"\n",(0,s.jsxs)(n.p,{children:[(0,s.jsx)(n.em,{children:"(experimental)"})," In addition, when operating in a pull request context,\n",(0,s.jsx)(n.code,{children:"base_file"})," versions of the ",(0,s.jsx)(n.code,{children:"file"})," operations are available for accessing the\nfiles in the base branch of the pull request. The ",(0,s.jsx)(n.code,{children:"file"})," versions of the\noperations operate on the head (proposed changes) versions of the files in a\npull request context."]}),"\n",(0,s.jsxs)(n.p,{children:["In addition, most of the\n",(0,s.jsx)(n.a,{href:"https://www.openpolicyagent.org/docs/latest/policy-reference/#built-in-functions",children:"standard OPA functions are available in the Minder runtime"}),"."]}),"\n",(0,s.jsx)(n.h2,{id:"example-codeql-enabled-check",children:"Example: CodeQL-enabled check"}),"\n",(0,s.jsx)(n.p,{children:"CodeQL is a very handy tool that GitHub provides to do static analysis on\ncodebases. In this scenario, we'll see a rule type that verifies that it's\nenabled via a GitHub action in the repository."}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-yaml",children:"---\nversion: v1\ntype: rule-type\nname: codeql_enabled\ncontext:\n  provider: github\ndescription: Verifies that CodeQL is enabled for the repository\nguidance: |\n  CodeQL is a tool that can be used to analyze code for security vulnerabilities.\n  It is recommended that repositories have some form of static analysis enabled\n  to ensure that vulnerabilities are not introduced into the codebase.\n\n  To enable CodeQL, add a GitHub workflow to the repository that runs the\n  CodeQL analysis.\n\n  For more information, see\n  https://docs.github.com/en/code-security/secure-coding/automatically-scanning-your-code-for-vulnerabilities-and-errors/configuring-code-scanning#configuring-code-scanning-for-a-private-repository\ndef:\n  # Defines the section of the pipeline the rule will appear in.\n  # This will affect the template used to render multiple parts\n  # of the rule.\n  in_entity: repository\n  # Defines the schema for writing a rule with this rule being checked\n  rule_schema:\n    type: object\n    properties:\n      languages:\n        type: array\n        items:\n          type: string\n        description: |\n          Only applicable for remediation. Sets the CodeQL languages to use in the workflow.\n          CodeQL supports 'c-cpp', 'csharp', 'go', 'java-kotlin', 'javascript-typescript', 'python', 'ruby', 'swift'\n      schedule_interval:\n        type: string\n        description: |\n          Only applicable for remediation. Sets the schedule interval for the workflow.\n    required:\n      - languages\n      - schedule_interval\n  # Defines the configuration for ingesting data relevant for the rule\n  ingest:\n    type: git\n    git:\n      branch: main\n  # Defines the configuration for evaluating data ingested against the given profile\n  eval:\n    type: rego\n    rego:\n      type: deny-by-default\n      def: |\n        package minder\n\n        default allow := false\n\n        allow {\n            # List all workflows\n            workflows := file.ls(\"./.github/workflows\")\n\n            # Read all workflows\n            some w\n            workflowstr := file.read(workflows[w])\n\n            workflow := yaml.unmarshal(workflowstr)\n\n            # Ensure a workflow contains the codel-ql action\n            some i\n            steps := workflow.jobs.analyze.steps[i]\n            startswith(steps.uses, \"github/codeql-action/analyze@\")\n        }\n  # Defines the configuration for alerting on the rule\n  alert:\n    type: security_advisory\n    security_advisory:\n      severity: 'medium'\n"})}),"\n",(0,s.jsxs)(n.p,{children:["The rego evaluation uses the ",(0,s.jsx)(n.code,{children:"deny-by-default"})," type. It'll set the policy as\nsuccessful if there is a GitHub workflow that instantiates\n",(0,s.jsx)(n.code,{children:"github/codeql-action/analyze"}),"."]}),"\n",(0,s.jsx)(n.h2,{id:"example-no-latest-tag-in-dockerfile",children:"Example: no 'latest' tag in Dockerfile"}),"\n",(0,s.jsxs)(n.p,{children:["In this scenario, we'll explore a rule type that verifies that a Dockerfile does\nnot use the ",(0,s.jsx)(n.code,{children:"latest"})," tag."]}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-yaml",children:"---\nversion: v1\ntype: rule-type\nname: dockerfile_no_latest_tag\ncontext:\n  provider: github\ndescription:\n  Verifies that the Dockerfile image references don't use the latest tag\nguidance: |\n  Using the latest tag for Docker images is not recommended as it can lead to unexpected behavior.\n  It is recommended to use a checksum instead, as that's immutable and will always point to the same image.\ndef:\n  # Defines the section of the pipeline the rule will appear in.\n  # This will affect the template used to render multiple parts\n  # of the rule.\n  in_entity: repository\n  # Defines the schema for writing a rule with this rule being checked\n  # In this case there are no settings that need to be configured\n  rule_schema: {}\n  # Defines the configuration for ingesting data relevant for the rule\n  ingest:\n    type: git\n    git:\n      branch: main\n  # Defines the configuration for evaluating data ingested against the given profile\n  # This example verifies that image in the Dockerfile do not use the 'latest' tag\n  # For example, this will fail:\n  # FROM golang:latest\n  # These will pass:\n  # FROM golang:1.21.4\n  # FROM golang@sha256:337543447173c2238c78d4851456760dcc57c1dfa8c3bcd94cbee8b0f7b32ad0\n  eval:\n    type: rego\n    rego:\n      type: constraints\n      def: |\n        package minder\n\n        violations[{\"msg\": msg}] {\n          # Read Dockerfile\n          dockerfile := file.read(\"Dockerfile\")\n\n          # Find all lines that start with FROM and have the latest tag\n          from_lines := regex.find_n(\"(?m)^(FROM .*:latest|FROM --platform=[^ ]+ [^: ]+|FROM (?!scratch$)[^: ]+)( (as|AS) [^ ]+)?$\", dockerfile, -1)\n          from_line := from_lines[_]\n\n          msg := sprintf(\"Dockerfile contains 'latest' tag in import: %s\", [from_line])\n        }\n  # Defines the configuration for alerting on the rule\n  alert:\n    type: security_advisory\n    security_advisory:\n      severity: 'medium'\n"})}),"\n",(0,s.jsx)(n.p,{children:"This leverages the constraints Rego evaluation type, which will output a failure\nfor each violation that it finds. This is handy for usability, as it will tell\nus exactly the lines that are not in conformance with our rules."}),"\n",(0,s.jsx)(n.h2,{id:"example-security-advisories-check",children:"Example: security advisories check"}),"\n",(0,s.jsx)(n.p,{children:"This is a more complex example. Here, we'll explore a rule type that checks for\nopen security advisories in a GitHub repository."}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-yaml",children:'---\nversion: v1\ntype: rule-type\nname: no_open_security_advisories\ncontext:\n  provider: github\ndescription: |\n  Verifies that a repository has no open security advisories based on a given severity threshold.\n\n  The threshold will cause the rule to fail if there are any open advisories at or above the threshold.\n  It is set to `high` by default, but can be overridden by setting the `severity` parameter.\nguidance: |\n  Ensuring that a repository has no open security advisories helps maintain a secure codebase.\n\n  This rule will fail if the repository has unacknowledged security advisories.\n  It will also fail if the repository has no security advisories enabled.\n\n  Security advisories that are closed or published are considered to be acknowledged.\n\n  For more information, see the [GitHub documentation](https://docs.github.com/en/code-security/security-advisories/working-with-repository-security-advisories/about-repository-security-advisories).\ndef:\n  in_entity: repository\n  rule_schema:\n    type: object\n    properties:\n      severity:\n        type: string\n        enum:\n          - unknown\n          - low\n          - medium\n          - high\n          - critical\n        default: high\n    required:\n      - severity\n  ingest:\n    type: rest\n    rest:\n      endpoint: \'/repos/{{.Entity.Owner}}/{{.Entity.Name}}/security-advisories?per_page=100&sort=updated&order=asc\'\n      parse: json\n      fallback:\n        # If we don\'t have advisories enabled, we\'ll get a 404\n        - http_code: 404\n          body: |\n            {"fallback": true}\n  eval:\n    type: rego\n    rego:\n      type: constraints\n      violation_format: json\n      def: |\n        package minder\n\n        import future.keywords.contains\n        import future.keywords.if\n        import future.keywords.in\n\n        severity_to_number := {\n          null: -1,\n          "unknown": -1,\n          "low": 0,\n          "medium": 1,\n          "high": 2,\n          "critical": 3,\n        }\n\n        default threshold := 1\n\n        threshold := severity_to_number[input.profile.severity] if input.profile.severity != null\n\n        above_threshold(severity, threshold) if {\n          severity_to_number[severity] >= threshold\n        }\n\n        had_fallback if {\n          input.ingested.fallback\n        }\n\n        violations contains {"msg": "Security advisories not enabled."} if {\n          had_fallback\n        }\n\n        violations contains {"msg": "Found open security advisories in or above threshold"} if {\n          not had_fallback\n\n          some adv in input.ingested\n\n          # Is not withdrawn\n          adv.withdrawn_at == null\n\n          adv.state != "closed"\n          adv.state != "published"\n\n          # We only care about advisories that are at or above the threshold\n          above_threshold(adv.severity, threshold)\n        }\n  alert:\n    type: security_advisory\n    security_advisory:\n      severity: \'medium\'\n'})}),"\n",(0,s.jsx)(n.p,{children:"This verifies that a repository does not have untriaged security advisories\nwithin a given severity threshold. Thus ensuring that the team is actively\ntaking care of the advisories and publishing or closing them depending on the\napplicability."}),"\n",(0,s.jsx)(n.h2,{id:"linting",children:"Linting"}),"\n",(0,s.jsxs)(n.p,{children:["In order to enforce correctness and best practices for our rule types, we have a\ncommand-line utility called\n",(0,s.jsx)(n.a,{href:"https://github.com/mindersec/minder/tree/main/cmd/dev",children:"mindev"})," that has a lint\nsub-command."]}),"\n",(0,s.jsx)(n.p,{children:"You can run it by doing the following from the Minder repository:"}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-bash",children:"./bin/mindev ruletype lint -r path/to/rule\n"})}),"\n",(0,s.jsx)(n.p,{children:"This will show you a list of suggestions to fix in your rule type definition."}),"\n",(0,s.jsxs)(n.p,{children:["The Styra team released a tool called\n",(0,s.jsx)(n.a,{href:"https://github.com/StyraInc/regal",children:"Regal"}),", which allows us to lint Rego\npolicies for best practices or common issues. We embedded Regal into our own\nrule linting tool within mindev. So, running ",(0,s.jsx)(n.code,{children:"mindev ruletype lint"})," on a rule\ntype that leverages Rego will also show you OPA-related best practices."]}),"\n",(0,s.jsx)(n.p,{children:"Conclusion"}),"\n",(0,s.jsx)(n.p,{children:"This introductory guide provides a foundation for leveraging Rego and Minder to\nwrite policies effectively. Experiment, explore, and tailor these techniques to\nmeet the unique requirements of your projects."}),"\n",(0,s.jsx)(n.p,{children:"Minder is constantly evolving, so don't be surprised if we soon add more custom\nfunctions or even more evaluation engines! The project is in full steam and more\nfeatures are coming!"}),"\n",(0,s.jsxs)(n.p,{children:["You can see a collection of rule types that we actively maintain in the\n",(0,s.jsx)(n.a,{href:"https://github.com/mindersec/minder-rules-and-profiles",children:"minder-rules-and-profiles repo"})]})]})}function h(e={}){const{wrapper:n}={...(0,r.R)(),...e.components};return n?(0,s.jsx)(n,{...e,children:(0,s.jsx)(d,{...e})}):d(e)}},28453:(e,n,i)=>{i.d(n,{R:()=>o,x:()=>a});var t=i(96540);const s={},r=t.createContext(s);function o(e){const n=t.useContext(r);return t.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function a(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:o(e.components),t.createElement(r.Provider,{value:n},e.children)}}}]);