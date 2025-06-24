# smarterr Diagnostics & Fallbacks

> **Template Types and Usage:**
>
> - `AddError` and `Append` use `error_summary` and `error_detail` templates (for Go errors).
> - `EnrichAppend` uses `diagnostic_summary` and `diagnostic_detail` templates (for framework diagnostics).
> - All output is a diagnostic; the template name refers to the input type (error vs. diagnostic).

smarterr is designed to never obscure the main error. If config is missing, broken, or a template fails, smarterr:

- Always includes the original error in the output.
- Appends a `[smarterr diagnostics]` section to the detail if internal errors occur.
- Logs all internal errors via `Debugf` if debug is enabled (see the [`smarterr` block](schema.md#smarterr-optional)).

---

## Fallback Behavior

- If config is missing or broken, the original error is shown.
- If a template or token cannot be resolved, a fallback message is used (see [`token_error_mode`](schema.md#smarterr-optional)).
- Internal errors (e.g., regex compile errors, template errors) are aggregated and shown in a `[smarterr diagnostics]` section.

---

## Debug Logging

Enable debug logging by setting `debug = true` in the `smarterr` block. All internal errors and fallbacks will be logged to stderr.

```hcl
smarterr {
  debug = true
}
```

---

## Example: Diagnostics Section

```text
ID: r-1234567890
Cause: operation error CloudWatch: ModifyServerlessCache
[smarterr diagnostics]
- template "error_detail" not found
- hint "foo" regex compile error: ...
```

---

## See Also

- [Full Config Schema](schema.md)
- [Layered Configs & Merging](layering.md)
