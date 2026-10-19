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
| **`stderr`** | Errors, warnings, interactive prompts, banners, and mutation notices | `cmd.ErrOrStderr()`, `cmd.PrintErr*` | Error logs, login prompts, browser URLs, Updated repository foo/bar |

---

## Detailed Rules

### 1. Standard Output (`stdout`)

`stdout` is strictly reserved for the primary data payload that the command produces. If a user pipes the output of a
command (e.g. `minder ... | jq` or `TOKEN=$(minder auth token)`), only valid payload data must appear on this stream.

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
- **Progress indicators and banners**: Step-by-step progress, spinners, and authentication status messages.
- **Action and mutation success notices**: Informational notices reporting state changes (e.g.,
  `Successfully created profile...`, `Updated repository...`, `Deleted data source...`). Following the convention
  of tools like GitHub CLI (`gh`), plain-text success messages are routed to `stderr` so scripts can rely on the
  process exit code (`0`) while capturing clean stdout payloads.

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
  payload (table, JSON, YAML, or raw value).
- **`ExpectedStderr`**: Asserts that `stderr` contains the given substring (checked via `assert.Contains`).
- **Verify stream isolation**: On error, nothing should be written to `stdout`. For read-only commands that succeed,
  `stderr` should normally be empty.
