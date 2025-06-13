# smarterr Layered Configs & Merging

smarterr supports layered, directory-based configuration. This allows you to define global, parent, and subdirectory configs that are automatically merged for each error site.

---

## How It Works

- **Discovery (Embedded):**
  - When using embedded configs (the most common case for providers/plugins), smarterr examines all files in the embedded filesystem whose name is `smarterr.hcl`.
  - For a given error site, it determines which embedded config files are "related" by comparing their paths to the call site (relative to the configured base directory).
  - All matching configs (from global to most specific) are loaded and merged.
  - **Global global config:** If `<base dir>/smarterr/smarterr.hcl` exists, it is always included first and acts as the most global config (even more global than a parent directory config).
  - The global config (at the base dir) is always included if present.
  - **Note:** smarterr does not walk the real filesystem at runtime; it operates on the set of embedded files. (The CLI will support real FS traversal in the future.)

- **Merging:**
  - Configs are merged from least to most specific (global global → global → parent → local).
  - For each block type, later (more specific) blocks override earlier ones by name.
  - The `smarterr` block is merged: most specific settings win.

---

## Example

```
project/
  smarterr.hcl         # global config
  subdir/
    smarterr.hcl      # subdir config (overrides or extends global)
```

If an error occurs in `subdir`, both configs are loaded and merged. Any block (token, parameter, etc.) defined in `subdir/smarterr.hcl` overrides the global one by name.

---

## Precedence Rules

- **Tokens, Hints, Parameters, StackMatches, Templates, Transforms:**
  - Merged by name; subdir overrides global.
- **smarterr block:**
  - Most specific (closest to error site) wins for each field.

---

## See Also
- [Full Config Schema](schema.md)
- [Diagnostics & Fallbacks](diagnostics.md)
