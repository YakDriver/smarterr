# smarterr

**Declarative, layered, and maintainable error messages for Go.**

With a single line of code:

```go
return smarterr.AppendSDK(ctx, diags, err, "id", "r-1234567890")
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
   smarterr.AppendFW(ctx, diags, err, "id", id)
   // or for SDK diagnostics:
   diags = smarterr.AppendSDK(ctx, diags, err, "id", id)
   ```

### CLI Usage (Work in Progress)

> **Note:** The smarterr CLI is under development and not yet available for use.
> When released, it will:
> - Output the effective merged configuration for any directory (after all layering/merging).
> - Validate all discovered configuration files for errors, missing fields, or schema issues.

Planned usage:

```sh
smarterr config --base-dir /path/to/project --start-dir /path/to/project/internal/service
smarterr validate --base-dir /path/to/project --start-dir /path/to/project/internal/service
```

---

## Example Configuration

Here’s a sample `smarterr.hcl` for a Terraform provider. For details, see the [full config schema](docs/schema.md).

```hcl
# 
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
