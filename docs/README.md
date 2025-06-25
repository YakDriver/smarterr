# smarterr documentation

Welcome to the documentation for **smarterr**—a declarative, layered, and maintainable error handling library for Go. smarterr lets you standardize, enrich, and centrally manage error messages and diagnostics across large codebases, all driven by configuration instead of scattered code changes.

## About smarterr

smarterr, a Go library:

- Centralizes error formatting and diagnostics using Config files (not code changes)
- Supports layered, directory-based configuration for scalable error management
- Produces actionable, user-friendly error summaries, details, and logs
- Never obscures the original error, even where you don't define or break Config

## Template types and usage

smarterr supports two main template types for customizing diagnostic output:

- **Error templates**: `error_summary` and `error_detail`
  - Used when formatting diagnostics from Go errors (for example, via `AddError` or `Append`).
- **Diagnostic templates**: `diagnostic_summary` and `diagnostic_detail`
  - Used when enriching framework-generated diagnostics (for example, via `EnrichAppend`).

> **Note:** Using `error_summary`, `error_detail`, `diagnostic_summary`, and `diagnostic_detail` smarterr outputs diagnostics; the template name refers to the input type (error vs. diagnostic).

**Function-to-template mapping:**

- `AddError` and `Append` use `error_summary` and `error_detail`.
- `EnrichAppend` uses `diagnostic_summary` and `diagnostic_detail`.

If you don't define the relevant templates, smarterr falls back to the original error or diagnostic content.

## Key features

- **Declarative error output:** Define error messages, details, and logs in Config files
- **Layered configs:** Merge global, parent, and local configs for flexible control
- **Diagnostic enrichment:** Enhance framework-generated diagnostics with context and suggestions
- **Safe fallbacks:** Always show the original error if Config or templates fail

## Where to start

Start with these docs:

- [**API Reference**](api.md):
  - How to use smarterr in your Go code (setup, error wrapping, appending, logger integration)
- [**Config Schema**](schema.md):
  - Full reference for all Config blocks, fields, and options in `smarterr.hcl`
- [**Layered configs & Merging**](layering.md):
  - How smarterr discovers, merges, and applies configs across directories
- [**Diagnostics & Fallbacks**](diagnostics.md):
  - How smarterr handles missing/broken Config to protect errors

## Example use cases

- Standardize error output for all resources in a Terraform provider
- Add suggestions to framework diagnostics without code changes
- Evolve error messages and formatting centrally, even in large codebases

## More information

- For a high-level overview and quickstart, see the root [README.md](../README.md)
- For advanced Config examples, see the [Config Schema](schema.md)

---

**smarterr** helps you deliver better error messages, faster diagnostics, and safer refactoring—at scale.
