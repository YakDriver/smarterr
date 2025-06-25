# smarterr diagnostics & fallbacks

> **Template types and usage:**
>
> - `AddError` and `Append` use `error_summary` and `error_detail` templates (for Go errors).
> - `EnrichAppend` uses `diagnostic_summary` and `diagnostic_detail` templates (for framework diagnostics).
> - Using `error_summary`, `error_detail`, `diagnostic_summary`, and `diagnostic_detail` smarterr outputs diagnostics; the template name refers to the input type (error vs. diagnostic).

smarterr never obscures the main error. If you don't define Config or make a mistake, smarterr:

- Always includes the original error in the output.
- Appends a `[smarterr diagnostics]` section to the detail if internal errors occur.
- Logs all internal errors via `Debugf` if you enable debugging (see the [`smarterr` block](schema.md#smarterr-optional)).

---

## Fallback behavior

- If you don't define Config or define broken Config, smarterr shows the original error.
- If smarterr can resolve a template or token, it uses a fallback message (see [`token_error_mode`](schema.md#smarterr-optional)).

---

## Debug logging

Enable debug logging by setting `debug = true` in the `smarterr` block. smarterr will log information, warnings, internal errors, and fallbacks to stdout.

```hcl
smarterr {
  debug = true
}
```

---

## Example: Diagnostics section

```text
ID: r-1234567890
Cause: operation error CloudWatch: ModifyServerlessCache
[smarterr diagnostics]
- template "error_detail" not found
- hint "foo" regex compile error: ...
```

---

## See also

- [Full Config Schema](schema.md)
- [Layered configs & Merging](layering.md)
