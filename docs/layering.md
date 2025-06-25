# smarterr layered configs & merging

> **Template types and usage:**
>
> - `AddError` and `Append` use `error_summary` and `error_detail` templates (for Go errors).
> - `EnrichAppend` uses `diagnostic_summary` and `diagnostic_detail` templates (for framework diagnostics).
> - All output is a diagnostic; the template name refers to the input type (error vs. diagnostic).

smarterr supports layered, directory-based configuration. This allows you to define global, parent, and subdirectory configs that are automatically merged for each error site.

---

## How it works

- **Discovery (Embedded):**
  - When using embedded configs (the most common case for providers/plugins), smarterr examines all files in the embedded filesystem whose name is `smarterr.hcl`.
  - For a given error site, it determines which embedded Config files are "related" by comparing their paths to the call site (relative to the configured base directory).
  - smarterr loads and merges all matching configs (from global to most specific).
  - **Global Config:** If `<base dir>/smarterr/smarterr.hcl` exists, it's always included first and acts as the most global Config (even more global than a parent directory Config).
  - The global Config (at the base dir) is always included if present.
  - **Note:** smarterr doesn't walk the real filesystem at runtime; it operates on the set of embedded files.

- **Merging:**
  - smarterr merges configs from least to most specific (global → parent → local). In other words, local takes precedence over parent or global configuration.
  - For each block type, later (more specific) blocks override earlier ones by name.

---

## Example

```text
project/
  smarterr.hcl        # parent config
  subdir/
    smarterr.hcl      # subdir config (overrides or extends parent)
```

If an error occurs in `subdir`, smarterr loads and merges both configs. Any block (token, parameter, etc.) defined in `subdir/smarterr.hcl` overrides the global one by name.

---

## Precedence rules

- **Tokens, Hints, Parameters, StackMatches, Templates, Transforms:**
  - Merged by name; subdir overrides global.
- **smarterr block:**
  - Most specific (closest to error site) wins for each field.

---

## See also

- [Full Config Schema](schema.md)
- [Diagnostics & Fallbacks](diagnostics.md)
