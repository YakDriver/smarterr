# smarterr

With a single line of code:

```go
return smarterr.AppendSDK(ctx, diags, err, "id", "r-1234567890")
```

smarterr uses declarative configuration and pulls information from `context.Context`, the call stack (to determine "creating"), a static parameter in the config, and an argument to the `AppendSDK()` call, resulting in this pretty output:

```
creating CloudWatch Composite Alarm (r-1234567890): operation error CloudWatch: ModifyServerlessCache
```

---

**smarterr** is a novel Go library and CLI for declarative, layered, and maintainable error message formatting. It lets you define how errors are rendered—across thousands of call sites—using configuration, not code changes. This means you can update, standardize, and improve error messages for both developers and users without refactoring your codebase.

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
//go:embed **/smarterr.hcl
var smarterrFiles embed.FS

smarterr.SetFS(&smarterr.EmbeddedFS{FS: smarterrFiles}, "dir/where/goembed/is/called/such/as/internal")
```

2. **Call smarterr in your error handling:**

```go
smarterr.AppendFW(ctx, diags, err, "id", id)
// or for SDK diagnostics:
diags = smarterr.AppendSDK(ctx, diags, err, "id", id)
```

### CLI Usage

```sh
smarterr config --base-dir /path/to/project --start-dir /path/to/project/internal/service
smarterr validate --base-dir /path/to/project --start-dir /path/to/project/internal/service
```

---

## Example Configuration

Here’s a sample `smarterr.hcl` for a Terraform provider:

```hcl
template "error_summary" {
  format = "{{.happening}} {{.service}} {{.resource}} ({{.identifier}}): {{.error}}"
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

## Example: Layered Configs (Parent and Subdirectory)

smarterr supports layered configuration. For example, you can have a parent config:

```hcl
# project/smarterr.hcl
token "service" {
  parameter = "service"
}

parameter "service" {
  value = "Default"
}
```

And in a subdirectory, you can add or override config:

```hcl
# project/subdir/smarterr.hcl
# This config can add tokens, templates, or override parameters from the parent
# For example, it can override the parent 'service' parameter automatically.
parameter "service" {
  value = "CloudWatch"
}
```

When smarterr loads config for a file in `subdir`, it merges both configs, so the `service` parameter will be used in the token (and template) inherited from the parent.

---

## Embedded vs. Real Filesystem

- **Embedded FS (recommended for libraries/plugins):**
  - Use Go’s `embed.FS` to bundle all your `smarterr.hcl` files.
  - Pass an embedded FS to `smarterr.SetFS()`.
- **Real FS (for CLI/debugging):**
  - Use `os.DirFS` and pass it to `smarterr.SetFS()` or use the CLI directly.
  - The CLI will walk up from `--start-dir` to `--base-dir` to discover configs.

---

## How AppendFW/AppendSDK Work

- `AppendFW(ctx, diags, err, ...)` — For Terraform Plugin Framework diagnostics.
- `AppendSDK(ctx, diags, err, ...)` — For Terraform Plugin SDK diagnostics.
- Both use the current call stack, context, and config to render a user-facing error message and add it to diagnostics.
- If config or rendering fails, smarterr always falls back to a safe, raw error message—never panicking or hiding the original error.

---

## FAQ

**Q: Do end users need to configure anything?**  
A: No. All config is embedded by the developer. End users get better errors automatically.

**Q: Can I update error messages without code changes?**  
A: Yes! Just update your config and re-embed.

**Q: What if config is missing or broken?**  
A: smarterr falls back to the original error, never panics, and logs the issue if debug is enabled.
