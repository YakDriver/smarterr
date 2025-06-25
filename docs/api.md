# smarterr library reference

This document describes the public Go API for smarterr, including configuration, logger setup, error wrapping/annotation, and error appending.

---

## Template types and usage

smarterr supports two main template types for customizing diagnostic output:

- **Error templates**: `error_summary` and `error_detail`
  - Used when formatting diagnostics from Go errors (for example, via `AddError` or `Append`).
- **Diagnostic templates**: `diagnostic_summary` and `diagnostic_detail`
  - Used when enriching framework-generated diagnostics (for example, via `EnrichAppend`).

> **Note:** All output produces a diagnostic. The template name refers to the input type (error vs. diagnostic).

**Function-to-template mapping:**

- `AddError` and `Append` use `error_summary` and `error_detail`.
- `EnrichAppend` uses `diagnostic_summary` and `diagnostic_detail`.

If you forget to define a template, smarterr falls back to the original error or diagnostic content.

---

## Filesystem setup

smarterr uses a virtual filesystem for Config discovery. You must set this at startup:

```go
func SetFS(fs FileSystem, baseDir string)
```

- `fs`: A filesystem implementation (for example, `*WrappedFS`).
- `baseDir`: The root directory for Config discovery (relative to embedded files or real FS).

### Embedded Config example (recommended for providers/plugins)

In a file called, for example, `internal/service/embed.go`:

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

- You can use several `//go:embed` lines to include files or patterns.
- `go:embed` **doesn't** recursively embed subdirectories; you must add a pattern for each depth you want (for example, `service/*/smarterr.hcl`, `service/*/*/smarterr.hcl`).
- Go resolves embedded files at compile time and includes them in the binary. **Config changes don't require code changes, but do require a new build.**

### Real filesystem example

```go
var fs = NewWrappedFS(os.DirFS("/path/to/configs"))
smarterr.SetFS(fs, "/path/to/configs")
```

---

## Logger setup

smarterr emits user-facing logs (not internal debug logs) via a pluggable logger interface. Set the logger at startup:

```go
func SetLogger(logger Logger)
```

### Provided loggers

- **TFLogLogger**: Integrates with Terraform's [`tflog`](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-log/tflog)

  ```go
  smarterr.SetLogger(smarterr.TFLogLogger{})
  ```

- **StdLogger**: Uses Go's standard `log` package.

  ```go
  smarterr.SetLogger(smarterr.StdLogger{})
  ```

You can create your own `Logger` if needed:

```go
type Logger interface {
    Debug(ctx context.Context, msg string, keyvals map[string]any)
    Info(ctx context.Context, msg string, keyvals map[string]any)
    Warn(ctx context.Context, msg string, keyvals map[string]any)
    Error(ctx context.Context, msg string, keyvals map[string]any)
}
```

---

## Error wrapping & annotation

smarterr provides structured error wrapping to capture context and call stack information at the point your application creates an error. This approach enables powerful, Config-driven diagnostics and stack matching.

### NewError

```go
func NewError(err error) error
```

Wraps an existing error with smarterr metadata, including a captured call stack. Use this at the site where an error first appears.

### Errorf

```go
func Errorf(format string, args ...any) error
```

Formats a new error (like `fmt.Errorf`) and captures the call stack and message. Use this for new errors.

#### Errorf example usage

```go
if err != nil {
    return smarterr.NewError(err)
}

return smarterr.Errorf("unexpected result for alarm %q", name)
```

You can pass the resulting error directly to `smarterr.Append` or `smarterr.AddError` for Config-driven formatting and diagnostics. smarterr uses the captured stack for advanced stack matching and template tokens.

### Error type

```go
type Error struct {
    Err         error             // The original or wrapped error
    Message     string            // Optional developer-provided message (from Errorf)
    Annotations map[string]string // Arbitrary key-value annotations (for example, subaction, resource_id)
    Stack       []runtime.Frame   // Captured call stack for stack matching
}
```

---

## Error appending

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

### EnrichAppend

```go
func EnrichAppend(ctx context.Context, existing *fwdiag.Diagnostics, incoming fwdiag.Diagnostics, keyvals ...any)
```

Enriches a set of framework diagnostics (`incoming`) with smarterr configuration and appends the enriched diagnostics to `existing` (mutating in place via pointer). Use it to enhance framework-generated diagnostics (such as value conversion errors) with context, suggestions, or improved formatting, all driven by Config.

- **Templates used:** `diagnostic_summary` and `diagnostic_detail` (if defined in Config)
- smarterr passes through the original diagnostic summary and detail if you don't define the templates.
- All output produces a diagnostic; the template name refers to the input type (diagnostic).

**Example usage:**

```go
smarterr.EnrichAppend(ctx, &resp.Diagnostics, incoming, smarterr.ID, req.Identity.Raw)
```

See also: [Template types and usage](#template-types-and-usage)

---

## Arguments

- `ctx`: Context for token resolution and logging.
- `diags`: Diagnostics object to append to.
- `err`: The error to format.
- `keyvals`: Optional key-value pairs for tokens.

---

## Reserved template names

smarterr uses special template names in your Config to control where output goes:

- `error_summary`: Rendered to the diagnostics summary (the main error message users see).
- `error_detail`: Rendered to the diagnostics detail (the expanded/collapsed error details).
- `diagnostic_summary`: Rendered to the diagnostics summary (the main error message users see).
- `diagnostic_detail`: Rendered to the diagnostics detail (the expanded/collapsed error details).
- `log_error`, `log_warn`, `log_info`: Rendered to the user-facing logger (for example, tflog or Go log) at the corresponding level.

> **Note:** smarterr outputs diagnostics. The template name refers to the input type (error vs. diagnostic).

You reference these templates by name in your Config. smarterr will automatically use them when you call `Append`, `AddError`, and `EnrichAppend`.

### Example Config and call

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
- The logger (for example, tflog) will receive the output of `log_error`.

See [Full Config Schema](schema.md) for all template and token options.

---

## Diagnostic token support

smarterr supports Config-driven enrichment of both errors and framework-generated diagnostics (for example, value conversion errors in Terraform Plugin Framework) via a special diagnostic token source.

### Diagnostic token

- Use `source = "diagnostic"` in a token block to expose a structured token with fields (for example, `.diag.summary`, `.diag.detail`, `.diag.severity`).
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

#### Template example

```hcl
template "diagnostic_summary" {
  format = "{{.happening}} {{.service}} {{.resource}}: {{.diag.summary}}"
}

template "diagnostic_detail" {
  format = "ID: {{.identifier}}\nCause: {{.diag.detail}}"
}
```

- smarterr populates the diagnostic token from the runtime (for example, framework diagnostic context) and can enrich and transform it via Config.

---

## Assert

```go
func Assert[T any](val T, err error) (T, error)
```

A helper for wrapping errors at the point of return. This wraps non-nil errors with `NewError` (capturing stack and context). Otherwise, it returns the value and error unchanged. This helper enables concise error handling in Go code.

### Assert example usage

```go
val, err := smarterr.Assert(doSomething())
if err != nil {
    return val, err
}

// Also
return smarterr.Assert(doSomething())
```

---

## Constants

smarterr provides several convenience constants for use in your code and configuration. These help standardize key names and reduce typos when referencing common tokens in templates or when passing key-value pairs to smarterr functions.

````go
const (
    ID           = "id"            // Standard key for resource or object identifier
    ResourceName = "resource_name" // Standard key for resource name
    ServiceName  = "service_name"  // Standard key for service name
)
````

You can use these constants when passing key-value pairs to `AddError`, `Append`, or `EnrichAppend`, or when defining tokens in your Config files. For example:

```go
smarterr.AddError(ctx, diags, err, smarterr.ID, id)
```

Or in your Config:

```hcl
token "identifier" {
  arg = "id"
}
```

---

## See also

- [Quickstart in README](../README.md#quickstart)
- [Full Config Schema](schema.md)
