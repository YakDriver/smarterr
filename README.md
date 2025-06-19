# smarterr

**Declarative, layered, and maintainable error messages for Go.**

With a single line of code:

```go
return smarterr.Append(ctx, diags, err, "id", "r-1234567890")
```

smarterr uses configuration—not code changes—to split an incoming error into **three output channels**:

1. **Error summary** (for users)

   ```
   creating CloudWatch Composite Alarm
   ```

2. **Error detail with a suggested fix** (for users)

   ```
   ID: r-1234567890
   Cause: operation error CloudWatch: ModifyServerlessCache
   If you are trying to modify a serverless cache, please use the
   `aws_cloudwatch_serverless_cache` resource instead of
   `aws_cloudwatch_log_group`.
   ```

3. **Log** ([`tflog`](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-log/tflog) or Go log)

   ```
   creating CloudWatch Composite Alarm (ID r-1234567890): operation error CloudWatch: ModifyServerlessCache, https response error StatusCode: 400, RequestID: 9c9c8b2c-d71b-4717-b62a-68cfa4b18aa9, InvalidParameterCombination: No
   ```

---

**smarterr** lets you define, update, and standardize error output for thousands of call sites—using config, not code. Evolve your error messages and formatting without cross-codebase refactors. Both developers and users get cleaner, more actionable diagnostics.

## smarterr: Library and CLI

smarterr consists of two components:

- **Go Library** – Integrate into your application or provider code to format errors, emit diagnostics, and control runtime logging.
- **Command-Line Tool (CLI)** – Use during development or CI to validate configuration files, inspect merged output, and catch issues early.

**Use the library** to power smarterr behavior at runtime.  
**Use the CLI** to debug and verify configs before they ship.

---

## Installation (CLI)

You can install the smarterr CLI with Go:

```sh
go install github.com/YakDriver/smarterr/cmd/smarterr@latest
```

This will install the `smarterr` binary in your `$GOPATH/bin` or `$HOME/go/bin`.

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

## Why smarterr?

- **For developers:**
  - No more hunting down and updating error messages in thousands of places.
  - Consistent, high-quality error output everywhere, driven by config.
  - Layered, declarative configuration—like you expect from modern tools—applied to error handling as a library.
  - Evolve your error messages and formatting without cross-codebase refactors.
- **For users:**
  - Cleaner, more helpful, and consistent error messages.
  - Context-aware diagnostics that are easier to understand and act on.

> In a large project (like the Terraform AWS Provider, with ~4000 error sites), smarterr lets you manage error output centrally, declaratively, and safely—an evolutionary step toward declarative development.

---

## Quickstart

### Library Usage

1. **Embed your config:**

  ```go
  //go:embed service/smarterr.hcl
  //go:embed service/*/smarterr.hcl
  var SmarterrFS embed.FS

  var smarterrInitOnce sync.Once

  func init() {
    smarterrInitOnce.Do(func() {
      smarterr.SetLogger(smarterr.TFLogLogger{})
      smarterr.SetFS(&smarterr.WrappedFS{FS: &SmarterrFS}, "dir/where/files/are/embedded/such/as/internal")
    })
  }
  ```

2. **Call smarterr in your error handling:**

   ```go
   smarterr.AddError(ctx, diags, err, "id", id) // uses error_summary/error_detail
   // or for SDK diagnostics:
   diags = smarterr.Append(ctx, diags, err, "id", id) // uses error_summary/error_detail
   // or to enrich framework diagnostics:
   smarterr.EnrichAppend(ctx, &diags, incoming, "id", id) // uses diagnostic_summary/diagnostic_detail
   ```

### CLI Usage

The smarterr CLI lets you validate and inspect your configuration files. It is stable and ready for use.

#### Commands

- **Show the effective merged configuration:**

  ```sh
  smarterr config --base-dir /path/to/project --start-dir /path/to/project/internal/service
  # or with short flags:
  smarterr config -b /path/to/project -d /path/to/project/internal/service
  ```

  This prints the merged config (after layering/merging) that would apply at the given directory.

- **Validate your configuration:**

  ```sh
  smarterr validate --base-dir /path/to/project --start-dir /path/to/project/internal/service
  # or with short flags:
  smarterr validate -b /path/to/project -d /path/to/project/internal/service
  ```

  This checks for parse errors, missing fields, schema issues, and other problems. The exit code is non-zero if errors are found.

  **Flags:**
  - `--quiet`, `-q`: Only output errors (suppresses merged config and warnings)
  - `--silent`, `-S`: No output, only exit code (non-zero if errors)
  - `--debug`, `-D`: Enable debug output

---

## Example Configuration

Here’s a sample `smarterr.hcl` for a Terraform provider. For details, see the [full config schema](docs/schema.md).

```hcl
# 
template "error_summary" {
  format = "{{.happening}} {{.service}} {{.resource}} ({{.identifier}}): {{.error}}"
}

template "diagnostic_summary" {
  format = "{{.happening}} {{.service}} {{.resource}}: {{.diag.summary}}"
}

token "happening" {
  stack_matches = [
    "create",
  ]
}

token "service" {
  parameter = "service"
}

token "resource" {
  context = "resource_name"
}

token "identifier" {
  arg = "id"
}

token "error" {
  source = "error"
  transforms = [
    "clean_aws_error"
  ]
}

token "diag" {
  source = "diagnostic"
  field_transforms = {
    summary = ["upper"]
    detail  = ["lower"]
  }
}

stack_match "create" {
  called_from = "resource[a-zA-Z0-9]*Create"
  display     = "creating"
}

parameter "service" {
  value = "CloudWatch"
}

transform "clean_aws_error" {
  step "remove" {
    regex = "RequestID: [a-z0-9-]+,"
  }
  step "remove" {
    value = "InvalidParameterCombination: No"
  }
  step "remove" {
    value = "https response error StatusCode: 400"
  }
  step "strip_suffix" {
    value = ","
    recurse = true
  }
}
```

---

## Diagnostic Enrichment & Structured Tokens

smarterr supports config-driven enrichment of both errors and framework-generated diagnostics (such as value conversion errors in Terraform Plugin Framework) using a structured diagnostic token.

### Diagnostic Token Usage

- Define a token with `source = "diagnostic"` to expose a structured token with fields (e.g., `.diag.summary`, `.diag.detail`, `.diag.severity`).
- Use `field_transforms` to apply transforms to individual fields of the diagnostic token.

Example:

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

Example template:

```hcl
template "diagnostic_summary" {
  format = "{{.happening}} {{.service}} {{.resource}}: {{.diag.summary}}"
}

template "diagnostic_detail" {
  format = "ID: {{.identifier}}\nCause: {{.diag.detail}}"
}
```

This enables actionable, context-rich diagnostics for both errors and framework-generated issues, all managed declaratively via config.

---

## Learn More

- [Full Config Schema](docs/schema.md)
- [Layered Configs & Merging](docs/layering.md)
- [Diagnostics & Fallbacks](docs/diagnostics.md)
- [API Reference](docs/api.md)
- [FAQ](#faq)

---

## FAQ

**Q: Do end users need to configure anything?**  
A: No. All config is embedded by the developer. End users get better errors automatically.

**Q: Can I update error messages without code changes?**  
A: Yes! Just update your config and re-embed.

**Q: What if config is missing or broken?**  
A: smarterr falls back to the original error, never panics, and logs the issue if debug is enabled.

For more, see the [Diagnostics & Fallbacks](docs/diagnostics.md) doc.
