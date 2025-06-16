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

### `token`

Reference:

```
token "name" {
  parameter    = "..."   # Reference a parameter
  context      = "..."   # Pull from context.Context
  arg          = "..."   # Pull from AppendSDK/FW args
  source       = "..."   # "parameter" | "context" | "arg" | "error" | "call_stack" | "error_stack" | "hints"
  stack_matches = [ ... ] # Names of stack_match blocks
  transforms   = [ ... ] # Names of transform blocks
}
```

- `source = "call_stack"`: Uses the live stack at the point of error reporting.
- `source = "error_stack"`: Uses the stack captured at the point of error creation (via `NewError`/`Errorf`).

Example:

```hcl
token "happening" {
  stack_matches = [
    "create",
    "read",
    "update",
    "delete",
    "read_set",
    "read_find",
    "create_wait"
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

token "clean_error" {
  source = "error"
  transforms = [
    "clean_aws_error"
  ]
}

token "error" {
  source = "error"
}

token "suggest" {
  source = "hints"
}
```

### `parameter`

Reference:

```
parameter "name" {
  value = "..."
}
```

Example:

```hcl
parameter "service" {
  value = "CloudWatch"
}
```

### `hint`

Reference:

```
hint "name" {
  error_contains = "..."   # Substring match on error
  regex_match    = "..."   # Regex match on error
  suggestion     = "..."   # Text to show if matched
}
```

Example:

```hcl
hint "example_hint" {
  error_contains = "InvalidParameterCombination"
  suggestion     = "Check your AWS resource parameters."
}
```

### `stack_match`

Reference:

```
stack_match "name" {
  called_from  = "..."   # Regex for function name
  called_after = "..."   # Regex for previous function in stack
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

stack_match "read_set" {
  called_after = "Set"
  called_from  = "resource[a-zA-Z0-9]*Read"
  display      = "setting during read"
}

stack_match "read_find" {
  called_after = "find.*"
  called_from  = "resource[a-zA-Z0-9]*Read"
  display      = "finding during read"
}

stack_match "create_wait" {
  called_after = "wait.*"
  called_from  = "resource[a-zA-Z0-9]*Create"
  display      = "waiting during creation"
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
