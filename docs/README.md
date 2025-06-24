# smarterr Documentation

Welcome to the documentation for **smarterr**—a declarative, layered, and maintainable error handling library for Go. smarterr lets you standardize, enrich, and centrally manage error messages and diagnostics across large codebases, all driven by configuration instead of scattered code changes.

## What is smarterr?

smarterr is a Go library that:

- Centralizes error formatting and diagnostics using config files (not code changes)
- Supports layered, directory-based configuration for scalable error management
- Produces actionable, user-friendly error summaries, details, and logs
- Never obscures the original error, even if config is missing or broken

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

## Key Features

- **Declarative error output:** Define error messages, details, and logs in config files
- **Layered configs:** Merge global, parent, and local configs for flexible control
- **Diagnostic enrichment:** Enhance framework-generated diagnostics with context and suggestions
- **Safe fallbacks:** Always show the original error if config or templates fail

## Where to Start

If you're new to smarterr, start with these docs:

- [**API Reference**](api.md):
  - How to use smarterr in your Go code (setup, error wrapping, appending, logger integration)
- [**Config Schema**](schema.md):
  - Full reference for all config blocks, fields, and options in `smarterr.hcl`
- [**Layered Configs & Merging**](layering.md):
  - How smarterr discovers, merges, and applies configs across directories
- [**Diagnostics & Fallbacks**](diagnostics.md):
  - How smarterr handles missing/broken config and ensures errors are never lost

## Example Use Cases

- Standardize error output for all resources in a Terraform provider
- Add actionable suggestions to framework diagnostics without code changes
- Evolve error messages and formatting centrally, even in large codebases

## More Information

- For a high-level overview and quickstart, see the root [README.md](../README.md)
- For advanced config examples, see the [Config Schema](schema.md)

---

**smarterr** helps you deliver better error messages, faster diagnostics, and safer refactoring—at scale.
