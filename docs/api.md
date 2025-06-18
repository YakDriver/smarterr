# smarterr Go API Reference

This document describes the public Go API for smarterr, including configuration, logger setup, error wrapping/annotation, and error appending.

---

## Template Types and Usage

smarterr supports two main template types for customizing diagnostic output:

- **Error templates**: `error_summary` and `error_detail`
  - Used when formatting diagnostics from Go errors (e.g., via `AddError` or `Append`).
- **Diagnostic templates**: `diagnostic_summary` and `diagnostic_detail`
  - Used when enriching framework-generated diagnostics (e.g., via `EnrichAppend`).

> **Note:** All output is a diagnostic. The template name refers to the input type (error vs. diagnostic).

**Function-to-template mapping:**
- `AddError` and `Append` use `error_summary` and `error_detail`.
- `EnrichAppend` uses `diagnostic_summary` and `diagnostic_detail`.

If the relevant templates are not defined, smarterr falls back to the original error or diagnostic content.

---

## Filesystem Setup

smarterr uses a virtual filesystem for config discovery. You must set this at startup:

```go
func SetFS(fs FileSystem, baseDir string)
```
- `fs`: A filesystem implementation (usually `*WrappedFS`).
- `baseDir`: The root directory for config discovery (relative to embedded files or real FS).

### Embedded Config Example (Recommended for Providers/Plugins)

In a file called, e.g., `internal/service/embed.go`:

```go
import (
    "embed"
    "sync"
    "github.com/YakDriver/smarterr"
)

//go:embed service/smarterr.hcl
//go:embed service/*/smarterr.hcl
var SmarterrFS embed.FS

var smarterrInitOnce sync.Once

func init() {
    smarterrInitOnce.Do(func() {
        smarterr.SetLogger(smarterr.TFLogLogger{})
        smarterr.SetFS(&smarterr.WrappedFS{FS: &SmarterrFS}, "internal")
    })
}
```

**go:embed tips:**
- You can use multiple `//go:embed` lines to include multiple files or patterns.
- `go:embed` does **not** recursively embed subdirectories; you must add a pattern for each depth you want (e.g., `service/*/smarterr.hcl`, `service/*/*/smarterr.hcl`).
- Embedded files are resolved at compile time and included in the binary. **Config changes do not require code changes, but do require a new build.**

### Real Filesystem Example (for CLI/debugging)

```go
var fs = NewWrappedFS(os.DirFS("/path/to/configs"))
smarterr.SetFS(fs, "/path/to/configs")
```

---

## Logger Setup

smarterr emits user-facing logs (not internal debug logs) via a pluggable logger interface. Set the logger at startup:

```go
func SetLogger(logger Logger)
```

### Provided Loggers

- **TFLogLogger**: Integrates with Terraform's [`tflog`](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-log/tflog)
  ```go
  smarterr.SetLogger(smarterr.TFLogLogger{})
  ```
- **StdLogger**: Uses Go's standard `log` package.
  ```go
  smarterr.SetLogger(smarterr.StdLogger{})
  ```

You can implement your own `Logger` if needed:

```go
type Logger interface {
    Debug(ctx context.Context, msg string, keyvals map[string]any)
    Info(ctx context.Context, msg string, keyvals map[string]any)
    Warn(ctx context.Context, msg string, keyvals map[string]any)
    Error(ctx context.Context, msg string, keyvals map[string]any)
}
```

---

## Error Wrapping & Annotation

smarterr provides structured error wrapping to capture context and call stack information at the point an error is created. This enables powerful, config-driven diagnostics and stack matching.

### NewError

```go
func NewError(err error) error
```
Wraps an existing error with smarterr metadata, including a captured call stack. Use this at the site where an error is first returned or recognized.

### Errorf

```go
func Errorf(format string, args ...any) error
```
Formats a new error (like `fmt.Errorf`) and captures the call stack and message. Use this for new errors.

#### Example Usage

```go
if err != nil {
    return smarterr.NewError(err)
}

return smarterr.Errorf("unexpected result for alarm %q", name)
```

The resulting error can be passed directly to `smarterr.Append` or `smarterr.AddError` for config-driven formatting and diagnostics. The captured stack is used for advanced stack matching and template tokens.

### Error Type

```go
type Error struct {
    Err         error             // The original or wrapped error
    Message     string            // Optional developer-provided message (from Errorf)
    Annotations map[string]string // Arbitrary key-value annotations (e.g., subaction, resource_id)
    Stack       []runtime.Frame   // Captured call stack for stack matching
}
```

---

## Error Appending

### AddError

```go
func AddError(ctx context.Context, diags fwdiag.Diagnostics, err error, keyvals ...any)
```
Adds a formatted error to Terraform Plugin Framework diagnostics.

### Append

```go
func Append(ctx context.Context, diags sdkdiag.Diagnostics, err error, keyvals ...any) sdkdiag.Diagnostics
```
Adds a formatted error to Terraform Plugin SDK diagnostics and returns the updated diagnostics slice.

---

## Arguments
- `ctx`: Context for token resolution and logging.
- `diags`: Diagnostics object to append to.
- `err`: The error to format.
- `keyvals`: Optional key-value pairs for tokens.

---

## Reserved Template Names

smarterr uses special template names in your config to control where output goes:

- `error_summary`: Rendered to the diagnostics summary (the main error message users see).
- `error_detail`: Rendered to the diagnostics detail (the expanded/collapsed error details).
- `log_error`, `log_warn`, `log_info`: Rendered to the user-facing logger (e.g., tflog or Go log) at the corresponding level.

You reference these templates by name in your config. smarterr will automatically use them when you call `Append` or `AddError`.

### Example: API Call + Config

**Go code:**
```go
smarterr.Append(ctx, diags, err, "id", id)
```

**Config (HCL):**
```hcl
template "error_summary" {
  format = "creating {{.service}} {{.resource}}"
}

template "error_detail" {
  format = "ID: {{.identifier}}\nCause: {{.error}}\n{{.hints}}"
}

template "log_error" {
  format = "creating {{.service}} {{.resource}} (ID {{.identifier}}): {{.error}}"
}
```

- The summary in diagnostics will use `error_summary`.
- The detail in diagnostics will use `error_detail`.
- The logger (e.g., tflog) will receive the output of `log_error`.

See [Full Config Schema](schema.md) for all template and token options.

---

## Diagnostic Token Support

smarterr supports config-driven enrichment of both errors and framework-generated diagnostics (e.g., value conversion errors in Terraform Plugin Framework) via a special diagnostic token source.

### Diagnostic Token

- Use `source = "diagnostic"` in a token block to expose a structured token with fields (e.g., `.diag.summary`, `.diag.detail`, `.diag.severity`).
- Use `field_transforms` to apply transforms to individual fields of the diagnostic token.

#### Example

```hcl
token "diag" {
  source = "diagnostic"
  field_transforms = {
    summary = ["upper"]
    detail  = ["lower"]
  }
}
```

In your template, access fields as `{{.diag.summary}}`, `{{.diag.detail}}`, etc.

#### Template Example

```hcl
template "diagnostic_summary" {
  format = "{{.happening}} {{.service}} {{.resource}}: {{.diag.summary}}"
}

template "diagnostic_detail" {
  format = "ID: {{.identifier}}\nCause: {{.diag.detail}}"
}
```

- The diagnostic token is populated from the runtime (e.g., framework diagnostic context) and can be enriched and transformed via config.

---

## Assert

```go
func Assert[T any](val T, err error) (T, error)
```
A helper for wrapping errors at the point of return. If `err` is non-nil, it wraps it with `NewError` (capturing stack and context); otherwise, it returns the value and error as-is. This is especially useful for concise error handling in Go code.

#### Example Usage

```go
val, err := smarterr.Assert(doSomething())
if err != nil {
    return val, err
}

// Also
return smarterr.Assert(doSomething())
```

---

## See Also
- [Quickstart in README](../README.md#quickstart)
- [Full Config Schema](schema.md)
