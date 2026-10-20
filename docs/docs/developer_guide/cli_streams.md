---
title: CLI Output Stream Conventions
sidebar_label: CLI output streams
sidebar_position: 30
---

# CLI Output Stream Conventions

To ensure consistent behavior across all Minder CLI commands and support clean shell scripting
(such as piping, redirection, and variable assignment), Minder follows strict conventions for standard
output (`stdout`) and standard error (`stderr`).

When developing or updating CLI commands, adhere to the guidelines below.

## Summary of Conventions

| Stream | Intended Content | Cobra / Go API | Examples |
|---|---|---|---|
| **`stdout`** | Primary command results, payloads, and structured data | `cmd.OutOrStdout()` | JSON, YAML, table formatting, tokens, version details |
| **`stderr`** | Errors, warnings, interactive prompts, banners, mutation notices, and debug output | `cmd.ErrOrStderr()`, `cmd.PrintErr*` | Error logs, login prompts, browser URLs, Updated repository foo/bar, `-v` debug logs |

---

## Detailed Rules

### 1. Standard Output (`stdout`)

`stdout` is strictly reserved for the primary data payload that the command produces. The user should be able to pipe
the output of a command to other tools (e.g. `minder ... | jq` or `TOKEN=$(minder auth token)`) without needing to trim
or adjust formatting.

Include on `stdout`:
- **Structured data outputs**: JSON (`-o json`) and YAML (`-o yaml`) representations of resources.
- **Table outputs**: Formatted ASCII/Unicode tables (`-o table`).
- **Identifiers and credentials**: Dedicated values meant for scripting, such as the access token printed by
  `minder auth token`.
- **Diagnostic output payloads**: Raw version and user-agent information from `minder version`.

### 2. Standard Error (`stderr`)

`stderr` is used for auxiliary information, diagnostics, user interaction, and operational notices. Writing these
messages to `stderr` ensures they do not pollute machine-readable output on `stdout`.

Include on `stderr`:
- **Errors and warnings**: Execution failures, deprecation warnings, and Cobra usage-on-error output.
- **Interactive prompts and instructions**: Browser login URLs, device code prompts, port listening notices, and
  confirmation prompts (e.g. in `minder auth login` or `minder provider enroll`).
- **Progress indicators and signs of life**: Status messages for long-running operations. Output should remain readable
  and avoid excessive spew:
  - Provide at least one update (or update an existing terminal line) every 5–10 seconds during long operations.
  - Limit output to at most 1 line per resource or operation, and consider batching notifications if total output
    would exceed 10–20 lines.
- **Action and mutation success notices**: Informational notices reporting state changes (e.g.,
  `Successfully created profile...`, `Updated repository...`, `Deleted data source...`). These success notices are not
  intended to be machine-processed. Keeping them on `stderr` allows future enhancements to print machine-readable
  payloads on `stdout` without breaking existing workflows.
- **Verbose and debug logging**: Detailed diagnostic logs enabled via `-v` / `--verbose` or debug flags. Routine
  status output should remain concise, reserving detailed trace information for explicit debug mode.

---

## Implementation Guidelines

### Using Cobra Streams in Command Implementations

When writing command logic:

- **For `stdout`**: Write explicitly to `cmd.OutOrStdout()` using `fmt.Fprint*` or table helpers:
  ```go
  // Structured payload
  fmt.Fprintln(cmd.OutOrStdout(), jsonString)

  // Using table helpers
  t := table.New(table.Simple, layouts.Default, cmd.OutOrStdout(), headers)
  ```

- **Avoid `cmd.Print*` for payloads**:
  In Cobra, convenience methods like `cmd.Print()`, `cmd.Println()`, and `cmd.Printf()` delegate to `cmd.OutOrStderr()`,
  falling back to `os.Stderr` when no output stream is set. In unit tests where `SetOut` is configured, they write to
  the test's output buffer, masking stream bugs. Always use `fmt.Fprint*(cmd.OutOrStdout(), ...)` explicitly for payloads.

- **For `stderr`**: Use `cmd.ErrOrStderr()` or Cobra's built-in error printers (`cmd.PrintErr*`):
  ```go
  // For formatted errors
  cmd.PrintErrf("Error processing resource: %v\n", err)

  // For informational prompts
  cmd.PrintErrln("Opening browser to authenticate...")
  ```

- **Avoid `cmd.OutOrStderr()`**:
  In Cobra, `cmd.OutOrStderr()` returns the output writer (`out`) whenever one is set (for example, in tests where
  output is captured). This causes messages intended for `stderr` to inadvertently bleed into `stdout`. Always prefer
  `cmd.ErrOrStderr()` or `cmd.PrintErr*` for error and diagnostic streams.

---

## Testing Stream Separation

Minder's test harness in `internal/util/cli/testing.go` captures `stdout` and `stderr` into independent buffers
(`stdoutBuf` and `stderrBuf`) during test execution with `cli.RunCmdTests`.

When writing tests for CLI commands:
- **`GoldenFileName`**: Validates the expected contents of `stdout`. Golden files should only contain the clean
  payload (table, JSON, YAML, or raw value). When updating existing tests or creating new ones, golden files can be
  generated or updated automatically by running tests with the `-update` flag (e.g. `go test ./... -update`) rather
  than modifying files manually.
- **`ExpectedStderr`**: Asserts that `stderr` contains the given substring (checked via `assert.Contains`).
- **Verify stream isolation**: On error, nothing should be written to `stdout`. For read-only commands that succeed,
  `stderr` should normally be empty.
