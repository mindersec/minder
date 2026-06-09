# Minder Rule Testing Framework -- Design

**Author:** Krrish Biswas
**Mentor:** Evan Anderson
**Status:** Draft -- Week 1 Design

---

## Abstract

Minder's security policy rules are currently untestable without a live Minder
server, a real GitHub token, and network access to external APIs. This document
proposes a standalone rule testing framework that allows rule authors to verify
rule correctness offline, using mocked HTTP responses and inline file fixtures.
The framework is built on [Starlark](https://github.com/google/starlark-go)
embedded in Go, and integrates with Go's native `testing` package for reporting.
A dedicated binary (`ruletest`) discovers and executes `*.star` test files
recursively from a directory.

---

## Background

### How Minder rules work

A Minder rule type defines a security policy check for an entity (repository,
artifact, pull request). Each rule has four phases:

<!-- markdownlint-disable MD013 -->

```text
  ┌─────────┐     ┌──────────┐     ┌───────────┐     ┌─────────┐
  │ Ingest  │────▶│ Evaluate │────▶│ Remediate │────▶│  Alert  │
  └─────────┘     └──────────┘     └───────────┘     └─────────┘
```

<!-- markdownlint-enable MD013 -->

**Ingest** fetches data from an external source. Three ingest types exist:

| Type | Mechanism | Example |
|---|---|---|
| `rest` | HTTP GET to GitHub API | Branch protection settings |
| `git` | Clone repo, expose `file.ls()` / `file.read()` | GitHub Actions workflows |
| `datasource` | Named HTTP client callable from Rego | Repository rulesets API |

**Evaluate** runs either a `jq` expression (simple field comparison) or a
`Rego/OPA` policy against the ingested data.

**Datasources** are named HTTP clients declared in the rule definition. When
Rego calls
`minder.datasource.baselineghapi.branch_protection_status({...})`,
the datasource fetches from a real API endpoint defined in a separate datasource
YAML. Responses are returned to Rego wrapped in
`{"body": <payload>, "status": 200}`.

### The current testing gap

There is no way to test a rule without:

- A running Minder server
- A real GitHub API token
- Live network access

This means a contributor who writes a new rule type has no way to verify it
works before opening a PR. Debugging requires a full round-trip: deploy, wait
for evaluation, observe result, repeat. The datasource `{"body"/"status"}`
wrapper has caused multi-hour debugging sessions because rule authors assume
they receive the raw payload.

### Related repositories

- `mindersec/minder` -- Core server, `pkg/engine`, `internal/datasources`
- `mindersec/minder-rules-and-profiles` -- Rule type YAML files and profiles

---

## Goals and Non-Goals

### Goals

- Allow rule authors to run rule tests **locally**, with no network access and
  no credentials
- Support all three ingest types: REST, git, and datasources
- Integrate with Go's `testing` package for standard test output and CI
  reporting
- Make simple tests trivially easy to write (single-function test cases under
  20 lines)
- Make hard tests possible (multi-source mocking, parameterized fixtures)
- **Auto-discover** test files in a directory tree -- no manual test registration

### Non-Goals (v1)

- Remediation output testing (verifying auto-fix PR content)
- Alert output testing
- Testing Rego `error()` abort behavior
- Timeout simulation
- Becoming a `minder` subcommand in v1 -- ships as a standalone binary first

---

## Overview

```text
  ┌──────────────────────────────────────┐
  │    ruletest --dir ./rule-types       │
  └──────────────┬───────────────────────┘
                 │
                 ▼
  ┌──────────────────────────────────────┐
  │    Test Discovery                    │
  │    *.star file walker                │
  └──────────────┬───────────────────────┘
                 │
                 ▼
  ┌──────────────────────────────────────┐
  │    Module Loader                     │
  │    go.starlark.net ExecFile          │
  └──────────────┬───────────────────────┘
                 │
                 ▼
  ┌──────────────────────────────────────┐
  │    Predeclared Builtins              │
  │    eval / read_file / txtar /        │
  │    check / test / body / code        │
  └───┬──────────┬───────────┬───────────┘
      │          │           │
      ▼          ▼           ▼
  ┌────────┐ ┌────────┐ ┌────────────┐
  │  eval  │ │read_file│ │  check.*   │
  │        │ │+ txtar  │ │  test()    │
  └───┬────┘ └───┬────┘ └─────┬──────┘
      │          │             │
      ▼          ▼             ▼
  ┌────────┐ ┌────────┐ ┌────────────┐
  │pkg/    │ │Mock FS │ │Go testing.T│
  │engine  │ │io/fs.FS│ │Pass / Fail │
  │Evaluate│ │from    │ │Error       │
  │Offline │ │txtar   │ └────────────┘
  └───┬────┘ └────────┘
      │
      ▼
  ┌──────────────────────────────────────┐
  │    Mock HTTP Layer                   │
  │    http.RoundTripper intercept       │
  │                                      │
  │    - REST ingest calls               │
  │    - Datasource calls                │
  └──────────────────────────────────────┘
```

The test runner (`ruletest`) walks a directory tree, finds all `*.star` files,
and executes each using an embedded Starlark interpreter. A set of predeclared
Go builtins (`eval`, `read_file`, `txtar`, `check`, `test`) are injected into
each module's environment. The builtins delegate to the existing `pkg/engine`
evaluation pipeline -- the same jq and Rego engines used in production -- with
HTTP calls intercepted by a mock layer.

---

## Detailed Design

### DD1: Test File Format -- Starlark

After prototyping three formats (Hybrid DSL, txtar+TOML, Starlark), Starlark
is selected as the test file format. See
[Alternatives Considered](#alternatives-considered) for the comparison.

The critical advantage: a real programming language gives parameterized
fixtures, shared helpers, and composable defaults without inventing a custom
templating syntax.

**Wrapper functions replace `[defaults]` blocks:**

```python
# Default args carry shared setup -- tests only specify what varies
def workflow_rule(ref, files="check_pinned.txtar", entity=DEFAULT_ENTITY):
    fs = {}
    if files != None:
        fs = {k: v.format(ref=ref) for k, v in txtar(read_file(files)).items()}
    return eval(
        rule   = "actions_check_pinned_tags",
        entity = entity,
        files  = fs,
    )

# One template txtar, two test variants -- no duplication
def test_pinned():   check.eq(workflow_rule(PINNED_SHA).status, "pass")
def test_floating(): check.eq(workflow_rule("v4").status, "fail")
```

**txtar as file-bundling layer:**

```text
-- .github/workflows/ci.yml --
name: CI
on: [push]
jobs:
  build:
    steps:
      - uses: actions/checkout@{ref}
```

The `{ref}` placeholder is substituted via Starlark's `.format()` -- enabling
parameterized git fixture files without duplicating workflow content per test
case.

---

### DD2: Datasource Mocking Strategy

This is the most significant design decision in the framework.

**Two approaches:**

```text
  Approach A -- Abstract Box          Approach B -- HTTP-Level (recommended)
  ┌───────────────────────┐          ┌───────────────────────┐
  │ test mocks             │          │ test mocks             │
  │ datasource:name/method │          │ GET api.github.com/... │
  └──────────┬────────────┘          └──────────┬────────────┘
             │                                  │
             ▼                                  ▼
  ┌───────────────────────┐          ┌───────────────────────┐
  │ Mock Layer             │          │ Mock HTTP interceptor  │
  │ skips datasource       │          └──────────┬────────────┘
  └──────────┬────────────┘                      │
             │                                   ▼
             │                       ┌───────────────────────┐
             │                       │ Datasource definition  │
             │                       │ processes real response │
             │                       └──────────┬────────────┘
             ▼                                  ▼
  ┌───────────────────────┐          ┌───────────────────────┐
  │ Rego                   │          │ Rego                   │
  │ receives mocked payload│          │ receives datasource    │
  └───────────────────────┘          │ output                 │
                                     └───────────────────────┘
```

#### Approach A -- Abstract box mocking

The test mocks the *output* of the datasource call directly, bypassing the
datasource definition:

```python
mocks = {
    "datasource:baselineghapi/branch_protection_status": {
        "body": {"applied_rulesets": [{"type": "non_fast_forward"}]},
    },
}
```

- **Advantage:** Simple to write. No knowledge of the underlying API shape
  needed.
- **Disadvantage:** The datasource definition is never exercised. If the
  datasource has a bug (wrong endpoint, wrong field mapping), tests still pass.
  Authors also need to know the datasource's output shape, which is the
  `{"body"/"status"}` wrapper.

#### Approach B -- HTTP-level mocking (recommended)

The test mocks at the *HTTP boundary*. The datasource definition is loaded, and
its HTTP calls are intercepted. Rule authors write mocks against real GitHub API
shapes:

```python
# Test: classic branch protection blocks force pushes
mocks = {
    "GET https://api.github.com/repos/acme-corp/widgets/branches/main/protection":
        body({"allow_force_pushes": {"enabled": False}}),
    "GET https://api.github.com/repos/acme-corp/widgets/rules/branches/main":
        body([{"ruleset_source_type": "Repository", "type": "non_fast_forward"}]),
}
```

```python
# Test: branch not protected (separate test case -- different mock for same endpoint)
mocks = {
    "GET https://api.github.com/repos/acme-corp/widgets/branches/main/protection":
        code(404),
}
```

`body(payload)` and `code(status)` are Go-supplied Starlark builtins:

- `body(x)` -- returns a 200 response with payload `x`
- `code(n)` -- returns an empty response with HTTP status `n`

**Advantages:**

- The datasource definition is loaded and exercised.
- Authors can go directly from GitHub API docs to test mocks -- no knowledge of
  datasource internals needed.
- The `{"body"/"status"}` wrapper is applied by the datasource as in
  production, so authors don't need to think about it.

**Disadvantages:**

- Requires loading datasource definitions.
- Needs a discovery mechanism for where datasource YAMLs live.
- URL patterns need glob support for path params (e.g.
  `repos/*/branches/*`).

**Recommendation:** Approach B. The test exercises more of the real pipeline
and aligns test mocks with GitHub API documentation rather than internal Minder
abstractions.

**Open question:** How are datasource definitions discovered? Options:

1. Inferred from the rule YAML (which declares
   `data_sources: [{name: baselineghapi}]`)
2. Co-located YAML in the same directory
3. Declared explicitly in the test file

---

### DD3: Test Discovery and File Layout

#### File layout options

The current `minder-rules-and-profiles` structure places rule YAML files in
flat directories:

```text
rule-types/github/
├── branch_protection_allow_force_pushes.yaml
├── osps-ac-03-01.yaml
└── actions_check_pinned_tags.yaml
```

Two co-location options for test files:

**Option A -- Flat directory (alongside rule YAML):**

```text
rule-types/github/
├── branch_protection_allow_force_pushes.yaml
├── branch_protection_allow_force_pushes.star   # co-located
├── osps-ac-03-01.yaml
└── osps-ac-03-01.star
```

- **Advantage:** Easy to see if a rule has a test -- it's right next to the
  YAML.
- **Disadvantage:** Directory grows with two files per rule; no visual
  separation between "config" and "test".

**Option B -- Subdirectory for tests:**

```text
rule-types/github/
├── branch_protection_allow_force_pushes.yaml
├── osps-ac-03-01.yaml
└── tests/
    ├── branch_protection_allow_force_pushes.star
    └── osps-ac-03-01.star
```

- **Advantage:** Clear separation between rule definitions and test code.
  Easier to see test coverage at a glance.
- **Disadvantage:** Adding a test requires navigating to a different directory;
  rule and test are farther apart.

Avoid **one directory per rule** -- creates many tiny directories, especially
when rules have no test yet.

#### Discovery

```bash
ruletest --dir ./rule-types
```

Recursive walk finds all `*.star` files. The directory path acts as the
implicit test suite -- no explicit suite configuration in v1.

**Rule loading:** Test runner loads all `*.yaml` files in the same directory
(or parent directory if using `tests/` subdir) as the `*.star` file, then
filters by the name passed to `eval(rule="name")`. Duplicate rule names in the
same directory: "don't do that" is sufficient for v1.

---

### DD4: Test Case Declaration

Four approaches are under active consideration.

#### Approach A -- `test_*` function discovery

Runner scans module globals after execution for no-arg callables whose names
start with `test_`. Function name becomes the test identifier.

```python
def test_force_pushes_disabled():
    result = eval(rule="branch_protection_allow_force_pushes",
                  entity=DEFAULT_ENTITY,
                  mocks={ENDPOINT: body({"allow_force_pushes": {"enabled": False}})})
    check.eq(result.status, "pass")
```

- **Advantage:** Familiar (pytest, Go `TestXxx`). Function name is naturally
  the test name. Custom assertions are straightforward.
- **Disadvantage:** Implementation requires invoking Starlark functions after
  module definition with correct thread context. Table-driven tests require
  all cases inside one `test_*` function.

**Implementation note:** This requires using `go.starlark.net`'s `Value`
interface to scan module globals and call functions. Worth prototyping with a
small Go program before committing to this approach.

#### Approach B -- `test()` builtin with `expect`

Test evaluation is a side effect of module loading. `test()` wraps both rule
evaluation and result validation:

```python
test("force pushes disabled", "branch_protection_allow_force_pushes",
     mocks  = {ENDPOINT: body({"allow_force_pushes": {"enabled": False}})},
     entity = DEFAULT_ENTITY,
     expect = "pass")
```

- **Advantage:** Name is explicit. Integrates directly with
  `t.Run("name", ...)`. No post-load scanning needed.
- **Disadvantage:** `expect=` handles simple pass/fail but not complex
  assertions (like checking violation messages).

#### Approach C -- `test()` with `(label, got, want)` tuples

Separates `eval()` from assertion so that debugging output preserves both the
actual and expected values:

```python
result = eval("no_open_security_advisories", entity=DEFAULT,
              mocks={"/*": code(404)})

test("security advisories off", [
    ("status",          result.status,          "fail"),
    ("violation count", len(result.violations), 1),
    ("first violation", result.violations[0],   "Security advisories not enabled."),
])
```

On failure, the Go side has all three values and can print:

```text
FAIL: security advisories off
  violation count: got 3, want 1
  first violation: got "No SECURITY.md found", want "Security advisories not enabled."
```

- **Advantage:** The label is for humans, `got`/`want` are for the diff --
  nothing gets "eaten". Custom assertions on violations, counts, and messages
  are natural.
- **Disadvantage:** More verbose for simple pass/fail tests. Tuple syntax is
  less readable than a simple `expect="pass"`.

#### Approach D -- `cases` dict + comprehension

```python
def run_test(name, case):
    result = eval(rule="actions_check_pinned_tags",
                  entity=DEFAULT_ENTITY,
                  files=workflow_files(case["ref"]))
    test(name, [("status", result.status, case["expect"])])

cases = {
    "pinned":   {"ref": PINNED_SHA,   "expect": "pass"},
    "floating": {"ref": FLOATING_TAG, "expect": "fail"},
}
[run_test(k, v) for k, v in cases.items()]
```

- **Advantage:** Most compact for parametrized tests.
- **Disadvantage:** Dict key is the test name (data, not structure) -- harder
  to navigate.

**Current status:** No single approach is clearly best. All four will be
prototyped before week 2 narrowing. The `test()` builtin example shared during
design was illustrative, not prescriptive.

---

### DD5: Assertions -- Assert vs Expect Semantics

The `check` module needs a clear semantic contract. Per the
[Google Testing Blog](https://testing.googleblog.com/2008/07/tott-expect-vs-assert.html):

| Style | Behavior | Effect |
|---|---|---|
| **Assert** | Stops test on first failure | You see only the first error |
| **Expect** | Collects all failures, reports at end | You see all errors at once |

For a rule test framework, **expect semantics** are preferable -- a test with
multiple `check.*` calls should report all failures, not stop at the first.

**Proposed `check` module (expect-style):**

```python
check.eq(result.status, "pass")           # continue even if fails
check.eq(len(result.violations), 1)       # continue even if fails
check.contains(result.violations[0].msg, "unpinned")
```

All failures are collected and reported together when the test case completes.

**Implementation:** The `check` builtin maintains a thread-local error list
via `starlark.Thread`'s local storage. At test completion, the runner reads
this list and reports all collected failures.

#### Extended check helpers as a Starlark module

Rather than implementing every check variant as a Go builtin, a Starlark module
can define higher-level helpers that call the primitive `check.eq` and
`check.contains` builtins:

```python
# checks.star (shipped with ruletest, auto-loaded)
def violations_count(result, n):
    check.eq(len(result.violations), n)

def violation_contains(result, msg):
    found = False
    for v in result.violations:
        if msg in v.msg:
            found = True
    check.eq(found, True)
```

This means new check helpers can be added without Go changes. The core Go
builtins stay minimal (`check.eq`, `check.ne`, `check.contains`); the Starlark
module layer provides ergonomic wrappers.

---

### DD6: Go Builtins Implementation

All builtins use `go.starlark.net`'s `starlark.NewBuiltin` pattern. The
predeclared environment is passed to `starlark.ExecFile`:

```go
predeclared := starlark.StringDict{
    "eval":      makeEval(evaluator),
    "read_file": makeReadFile(testDir),   // sandboxed I/O
    "txtar":     makeTxtar(),             // pure string parser
    "check":     makeCheckModule(),       // expect-style assertions
    "body":      makeBody(),              // mock response helper
    "code":      makeCode(),              // mock error response helper
}

thread := &starlark.Thread{Name: filepath.Base(filename)}
globals, err := starlark.ExecFile(thread, filename, nil, predeclared)
```

#### `eval` -- rule evaluator

```go
// Starlark: eval(rule, entity, mocks?, files?) -> EvalResult
func makeEval(evaluator *engine.RuleEvaluator) starlark.Value {
    return starlark.NewBuiltin("eval", func(
        thread *starlark.Thread,
        b      *starlark.Builtin,
        args   starlark.Tuple,
        kwargs []starlark.Tuple,
    ) (starlark.Value, error) {
        var ruleName string
        var entity, mocks, files starlark.Value
        if err := starlark.UnpackArgs(b.Name(), args, kwargs,
            "rule",   &ruleName,
            "entity", &entity,
            "mocks?", &mocks,
            "files?", &files,
        ); err != nil {
            return nil, err
        }
        result, err := evaluator.EvaluateOffline(ruleName,
            toGoMap(entity), toMocks(mocks), toFileMap(files))
        if err != nil {
            return nil, err
        }
        return newEvalResult(result), nil
    })
}
```

#### `read_file` -- sandboxed file reader

```go
// Starlark: read_file(path) -> string
// Sandbox: reject absolute paths and ".." traversal
func makeReadFile(testDir string) starlark.Value {
    return starlark.NewBuiltin("read_file", func(
        _ *starlark.Thread, b *starlark.Builtin,
        args starlark.Tuple, kwargs []starlark.Tuple,
    ) (starlark.Value, error) {
        var path string
        if err := starlark.UnpackPositionalArgs(
            b.Name(), args, kwargs, 1, &path,
        ); err != nil {
            return nil, err
        }
        if filepath.IsAbs(path) || strings.Contains(path, "..") {
            return nil, fmt.Errorf(
                "read_file: %q not allowed (must be relative, no ..)", path,
            )
        }
        data, err := os.ReadFile(filepath.Join(testDir, path))
        if err != nil {
            return nil, fmt.Errorf("read_file: %w", err)
        }
        return starlark.String(data), nil
    })
}
```

#### `txtar` -- pure string parser (no I/O)

```go
// Starlark: txtar(string) -> dict[filename -> content]
func makeTxtar() starlark.Value {
    return starlark.NewBuiltin("txtar", func(
        _ *starlark.Thread, b *starlark.Builtin,
        args starlark.Tuple, kwargs []starlark.Tuple,
    ) (starlark.Value, error) {
        var content starlark.String
        if err := starlark.UnpackPositionalArgs(
            b.Name(), args, kwargs, 1, &content,
        ); err != nil {
            return nil, err
        }
        archive := txtar.Parse([]byte(content.GoString()))
        d := new(starlark.Dict)
        for _, f := range archive.Files {
            d.SetKey(starlark.String(f.Name), starlark.String(f.Data))
        }
        return d, nil
    })
}
```

Sandboxing uses `io/fs.FS` rooted at the test file's directory -- the same
abstraction already used by Minder's go-billy git client.

---

### DD7: Error Propagation

Starlark builtins return `(starlark.Value, error)`. An `error` return **stops
module execution** (assert semantics for the runtime). The `check` module uses
thread-local error collection to provide expect semantics at the test level:

```text
  Runner              Starlark            Check Module
    │                    │                     │
    │  ExecFile(test.star)                     │
    │───────────────────▶│                     │
    │                    │  check.eq("fail",   │
    │                    │           "pass")   │
    │                    │────────────────────▶│
    │                    │                     │ append error to
    │                    │                     │ thread-local list
    │                    │    return None      │
    │                    │◀────────────────────│
    │                    │                     │
    │                    │  check.contains(    │
    │                    │    msg, "unpinned")  │
    │                    │────────────────────▶│
    │                    │                     │ append error to
    │                    │                     │ thread-local list
    │                    │    return None      │
    │                    │◀────────────────────│
    │                    │                     │
    │   module complete  │                     │
    │◀───────────────────│                     │
    │                                          │
    │  read thread-local error list            │
    │─────────────────────────────────────────▶│
    │                                          │
    │  report all 2 failures                   │
    │                                          │
```

`eval()` errors (e.g. unmocked HTTP call, unknown rule name) return Go
`error`, which surfaces as a Starlark exception and stops the current function.
This is appropriate -- a broken test setup should fail loudly, not silently
continue.

---

### DD8: Standalone Binary (`ruletest`)

The test runner ships as a standalone binary, not as a `minder` subcommand,
for v1.

**Rationale:** The binary pulls in `pkg/engine`, Starlark, datasource
definitions, and all their dependencies. This is likely significantly larger
than the `minder` CLI binary. Keeping it separate avoids bloating `minder` and
allows faster iteration on the test tooling independently.

**Target timeline:** Working `ruletest` binary by **week 5**, not week 2.
Earlier weeks focus on the Starlark execution environment and `eval`
correctness.

```bash
# Discovery and execution
ruletest --dir ./rule-types

# Single file
ruletest --file rule-types/github/branch_protection_allow_force_pushes.star

# Output: standard Go test output
--- PASS: branch_protection_allow_force_pushes/test_force_pushes_disabled (0.03s)
--- FAIL: branch_protection_allow_force_pushes/test_force_pushes_enabled (0.02s)
    check.eq: got "pass", want "fail"
FAIL
```

Binary name candidates: `ruletest`, `mindev`. To be decided.

---

## Alternatives Considered

### A1: Hybrid DSL (`when/then` format)

```text
test "force_pushes_disabled":
  when:
    github_rest GET /repos/.../protection:
      status: 200
      body:
        allow_force_pushes:
          enabled: false
  then:
    result: pass
```

- **Pros:** Readable for non-programmers. Clear visual separation of inputs
  and assertions.
- **Cons:** Requires custom tokenizer and BNF grammar. No parameterization or
  loops. No sharing of test setup across cases. Every check variant requires a
  grammar change.

### A2: txtar + TOML

```toml
[[test]]
name = "force_pushes_disabled"
mock.method = "GET"
mock.path = "/repos/.../branches/main/protection"
mock.status = 200
mock.body.allow_force_pushes.enabled = false
expect.result = "pass"
```

- **Pros:** Standard parsers exist (`BurntSushi/toml`).
  `[[array.of.tables]]` gives clear test boundaries. TOML dotted-key notation
  (`body.allow_force_pushes.enabled = false`) is clean for shallow data.
- **Cons:** `[[array.of.tables]]` nesting becomes hard to follow when a test
  case has multiple mock sources. No parameterization. Mock data and test
  definition are in the same file, coupling "dumb data" with "active logic".

txtar remains useful as a **file-bundling layer** alongside Starlark for
git-ingest test fixtures.

### A3: Embedded real language (Python, Lua, JavaScript)

- **Python:** Familiar but heavy dependency. Non-trivial to sandbox.
- **Lua:** Lightweight but niche -- unfamiliar to most contributors.
- **JavaScript/Node:** Familiar but significant runtime complexity.
- **Starlark (chosen):** Designed for embedding in Go tools. Deterministic
  (no I/O, no randomness by default). `go.starlark.net` is mature (used by
  Bazel). Python-like syntax. Sandboxable.

### A4: Datasource abstract box mocking

Mock the output of the datasource call directly (see DD2 -- Approach A).
Simpler to write but doesn't exercise the datasource definition. The
`{"body"/"status"}` wrapper becomes a concern for test authors. Rejected in
favor of HTTP-level mocking.

---

## Open Questions

| # | Question | Status |
|---|---|---|
| Q1 | `test_*` vs `test()` vs tuples vs `cases` -- which approach ships first? | Open -- prototype all four |
| Q2 | How are datasource definitions discovered and loaded? | Open |
| Q3 | File layout: Option A (flat) vs Option B (`tests/` subdir)? | Open |
| Q4 | `check` module: ship as Go builtins or auto-loaded Starlark module? | Leaning Starlark module |
| Q5 | Binary name: `ruletest` vs `mindev`? | Open |
| Q6 | Glob pattern matching for URL mocks (`repos/*/branches/*`)? | Needed for Approach B |
