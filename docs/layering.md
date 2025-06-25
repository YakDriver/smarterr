# smarterr layered configs & merging

> **Template types and usage:**
>
> - `AddError` and `Append` use `error_summary` and `error_detail` templates (for Go errors).
> - `EnrichAppend` uses `diagnostic_summary` and `diagnostic_detail` templates (for framework diagnostics).
> - Using `error_summary`, `error_detail`, `diagnostic_summary`, and `diagnostic_detail` smarterr outputs diagnostics; the template name refers to the input type (error vs. diagnostic).

smarterr supports layered, directory-based configuration. This allows you to define global, parent, and subdirectory configs that smarterr merges for each error site.

---

## How it works

- **Discovery (Embedded):**
  - When using embedded configs (the most common case for providers/plugins), smarterr examines all `smarterr.hcl` files in the embedded filesystem.
  - For a given error site, it uses "related" embedded Config files, comparing their paths to the call site (using the configured directory).
  - smarterr loads and merges all matching configs (from global to most specific).
  - **Global Config:** If `<base dir>/smarterr/smarterr.hcl` exists, it's always included first and acts as the most global Config (even more global than a parent directory Config).
  - smarterr includes the global Config if present.
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
