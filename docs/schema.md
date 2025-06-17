# smarterr Config Schema

This document describes the full configuration schema for smarterr, including all supported blocks, fields, and options. Use this as a reference for authoring or reviewing your `smarterr.hcl` files.

---

## Call Stack Sources: Live vs Captured

smarterr supports two types of call stack sources for stack matching and tokens:

- **Live Call Stack**: The stack at the point where `AppendSDK`/`AppendFW` is called. Use with `source = "call_stack"`.
- **Captured Call Stack**: The stack captured at the point where `NewError` or `Errorf` is called. Use with `source = "error_stack"`.

This distinction allows you to match on either the reporting site or the original error site, enabling more precise and context-aware diagnostics.

### Example

```hcl
token "happening" {
  source = "call_stack"
  stack_matches = ["create", "read", "update", "delete"]
}

token "subaction" {
  source = "error_stack"
  stack_matches = ["wait", "find", "set"]
}
```

---

## Top-Level Blocks

- `smarterr` (optional): Behavioral settings for error formatting and diagnostics.
- `template`: Defines named templates for error summary, detail, and logs.
- `token`: Declares a value to be resolved for use in templates.
- `parameter`: Static values for tokens.
- `hint`: Suggestion logic for error messages.
- `stack_match`: Call stack matching rules.
- `transform`: Value transformation pipelines.

---

## Block Reference

### `smarterr` (optional)

Reference:

```
smarterr {
  debug            = false         # Enable internal debug logging
  token_error_mode = "empty"      # "empty" | "placeholder" | "detailed"
  hint_join_char   = "\n"         # String to join multiple hints (default: newline)
  hint_match_mode  = "all"        # "all" | "first" (default: all)
}
```

Example:

```hcl
smarterr {
  debug = true
  hint_match_mode = "first"
}
```

### `template`

Reference:

```
template "error_summary" {
  format = "...Go text/template..."
}

template "error_detail" {
  format = "..."
}

template "log_error" {
  format = "..."
}
```

Example:

```hcl
template "error_summary" {
  format = "{{.happening}} {{.service}} {{.resource}}"
}

template "error_detail" {
  format = "ID: {{.identifier}}\nUnderlying issue: {{.clean_error}}{{if .suggest}}\n{{.suggest}}{{end}}"
}

template "log_error" {
  format = "{{.happening}} {{.service}} {{.resource}} (ID {{.identifier}}): {{.error}}"
}
```

### Template Types

smarterr supports the following template types:

- `error_summary`: Rendered for error summary (main error message).
- `error_detail`: Rendered for error detail (expanded/collapsed details).
- `diagnostic_summary`: Rendered for framework/diagnostic summary (e.g., value conversion errors).
- `diagnostic_detail`: Rendered for framework/diagnostic detail.
- `log_error`, `log_warn`, `log_info`: Rendered to the user-facing logger at the corresponding level.

Reference:

```
template "diagnostic_summary" {
  format = "{{.happening}} {{.service}} {{.resource}}: {{.original_summary}}"
}

template "diagnostic_detail" {
  format = "ID: {{.identifier}}\nOriginal: {{.original_detail}}\nContext: {{.happening}} {{.service}} {{.resource}}"
}
```

### `token`

Reference:

```
token "name" {
  parameter    = "..."   # Reference a parameter
  context      = "..."   # Pull from context.Context
  arg          = "..."   # Pull from AppendSDK/FW args
  source       = "..."   # "parameter" | "context" | "arg" | "error" | "call_stack" | "error_stack" | "hints" | "diagnostic"
  stack_matches = [ ... ] # Names of stack_match blocks
  transforms   = [ ... ] # Names of transform blocks (applies to string tokens)
  field_transforms = {   # (optional) For structured tokens like diagnostic, apply transforms to fields
    summary  = ["upper"]
    detail   = ["lower"]
    # ...
  }
}
```

- `source = "call_stack"`: Uses the live stack at the point of error reporting.
- `source = "error_stack"`: Uses the stack captured at the point of error creation (via `NewError`/`Errorf`).
- `source = "diagnostic"`: Exposes a structured token with fields (e.g., `.diag.summary`, `.diag.detail`, `.diag.severity`).
- `field_transforms`: Map of field name to list of transform names, applied to each field of a structured token.

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

### `stack_match`

Reference:

```
stack_match "name" {
  called_from  = "..."   # Regex for function name
  display      = "..."   # Value to use if matched
}
```

Example:

```hcl
stack_match "create" {
  called_from = "resource[a-zA-Z0-9]*Create"
  display     = "creating"
}

stack_match "read" {
  called_from = "resource[a-zA-Z0-9]*Read"
  display     = "reading"
}

stack_match "update" {
  called_from = "resource[a-zA-Z0-9]*Update"
  display     = "updating"
}

stack_match "delete" {
  called_from = "resource[a-zA-Z0-9]*Delete"
  display     = "deleting"
}

stack_match "wait" {
  called_from = "wait.*"
  display     = "waiting during operation"
}

stack_match "find" {
  called_from = "find.*"
  display     = "finding during operation"
}

stack_match "set" {
  called_from = "Set"
  display     = "setting during operation"
}
```

### `transform`

Reference:

```
transform "name" {
  step "type" {
    value   = "..."   # For strip_prefix, strip_suffix, remove, replace
    regex   = "..."   # For remove, replace
    with    = "..."   # For replace
    recurse = true    # (optional) Apply repeatedly
  }
  # Supported step types: strip_prefix, strip_suffix, remove, replace, trim_space, fix_space, lower, upper
}
```

Example:

```hcl
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

## Notes
- All blocks can be layered and merged across directories.
- See [docs/layering.md](layering.md) for details on config discovery and merging.
- See [docs/diagnostics.md](diagnostics.md) for fallback and diagnostics behavior.
- For advanced stack matching, see the distinction between `call_stack` and `error_stack` sources above.
