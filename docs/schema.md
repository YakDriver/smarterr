# smarterr Config schema

This document describes the full configuration schema for smarterr, including all supported blocks, fields, and options. Use this as a reference for authoring or reviewing your `smarterr.hcl` files.

---

## Template types and usage

smarterr supports two main template types for customizing diagnostic output:

- **Error templates**: `error_summary` and `error_detail`
  - Used when formatting diagnostics from Go errors (for example, via `AddError` or `Append`).
- **Diagnostic templates**: `diagnostic_summary` and `diagnostic_detail`
  - Used when enriching framework-generated diagnostics (for example, via `EnrichAppend`).

> **Note:** Using `error_summary`, `error_detail`, `diagnostic_summary`, and `diagnostic_detail`, smarterr outputs diagnostics; the template name refers to the input type (error vs. diagnostic).

**Function-to-template mapping:**

- `AddError` and `Append` use `error_summary` and `error_detail`.
- `EnrichAppend` uses `diagnostic_summary` and `diagnostic_detail`.

If you don't define Config or make a mistake, smarterr falls back to the original error or diagnostic content.

---

## Call stack sources: Live vs captured

smarterr supports two types of call stack sources for stack matching and tokens:

- **Live Call Stack**: The stack at the point where a host application calls `Append`/`AddError`/`EnrichAppend`. Use with `source = "call_stack"`.
- **Captured Call Stack**: The stack captured at the point where a host application calls `NewError` or `Errorf`. Use with `source = "error_stack"`.

You can match on either the reporting site or the original error site, enabling more precise and context-aware diagnostics.

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

## Top-level blocks

- `smarterr` (optional): Behavioral settings for error formatting and diagnostics.
- `template`: Defines named templates for error summary, detail, and logs.
- `token`: Declares a value smarterr will resolve for use in templates.
- `parameter`: Static values for tokens.
- `hint`: Suggestion logic for error messages.
- `stack_match`: Call stack matching rules.
- `transform`: Value transformation pipelines.

---

## Block reference

### `smarterr` (optional)

Reference:

```hcl
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

```hcl
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

### Template types

smarterr supports the following template types:

- `error_summary`: Rendered for error summary (main error message).
- `error_detail`: Rendered for error detail (expanded/collapsed details).
- `diagnostic_summary`: Rendered for framework/diagnostic summary (for example, value conversion errors).
- `diagnostic_detail`: Rendered for framework/diagnostic detail.
- `log_error`, `log_warn`, `log_info`: Rendered to the user-facing logger at the corresponding level.

Reference:

```hcl
template "diagnostic_summary" {
  format = "{{.happening}} {{.service}} {{.resource}}: {{.original_summary}}"
}

template "diagnostic_detail" {
  format = "ID: {{.identifier}}\nOriginal: {{.original_detail}}\nContext: {{.happening}} {{.service}} {{.resource}}"
}
```

### `token`

Reference:

```hcl
token "name" {
  parameter    = "..."   # Reference a parameter
  context      = "..."   # Pull from context.Context
  arg          = "..."   # Pull from Append/AddError args
  source       = "..."   # "parameter" | "context" | "arg" | "error" | "call_stack" | "error_stack" | "hints" | "diagnostic"
  stack_matches = [ ... ] # Names of stack_match blocks
  transforms   = [ ... ] # Names of transform blocks (applies to the whole token value)
  field_transforms = {   # (optional) For structured tokens (like diagnostic), apply transforms to specific fields
    summary  = ["upper"]
    detail   = ["lower"]
    # ...
  }
}
```

- `source = "call_stack"`: Uses the live stack at the point of error reporting.
- `source = "error_stack"`: Uses the stack captured at the point of error creation (via `NewError`/`Errorf`).
- `source = "diagnostic"`: Exposes a structured token with fields (for example, `.diag.summary`, `.diag.detail`, `.diag.severity`).
- `transforms`: In order, applies the listed transforms to the entire value of the token. Use this for string tokens.
- `field_transforms`: Applies the listed transforms to specific fields of a structured token (such as one with `source = "diagnostic"`). Use this when the token resolves to a map/object and you want to transform fields differently.

**Distinction:**

- Use `transforms` for simple tokens (single string value).
- Use `field_transforms` for structured tokens (map/object), for example, diagnostic tokens with fields like `summary`, `detail`, etc.
- You can define both but `field_transforms` applies to structured tokens.

Example:

```hcl
token "resource" {
  transforms = ["lower", "trim_space"]
}

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

```hcl
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

```hcl
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

#### Supported transform types

Transform step types, what they do, and example usages:

---

#### `strip_prefix`

Removes a specified prefix from the beginning of the value. If `recurse = true`, smarterr will remove the prefix until it can't find a match.

**Example:**

```hcl
transform "remove_prefix" {
  step "strip_prefix" {
    value = "ERR: "
    recurse = true
  }
}
```

- Input: `"ERR: ERR: Something went wrong"`
- Output: `"Something went wrong"`

---

#### `strip_suffix`

Removes a specified suffix from the end of the value. If `recurse = true`, smarterr will remove the suffix until it can't find a match.

**Example:**

```hcl
transform "remove_trailing_comma" {
  step "strip_suffix" {
    value = ","
    recurse = true
  }
}
```

- Input: `"foo,bar,baz,,"`
- Output: `"foo,bar,baz"`

---

#### `remove`

Removes all occurrences of a substring (`value`) or a regular expression match (`regex`). If `recurse = true`, smarterr will remove until it can't find any matches.

**Example (substring):**

```hcl
transform "remove_word" {
  step "remove" {
    value = "DEBUG"
    recurse = true
  }
}
```

- Input: `"DEBUG: something DEBUG happened"`
- Output: `": something  happened"`

**Example (regex):**

```hcl
transform "remove_digits" {
  step "remove" {
    regex = "[0-9]+"
  }
}
```

- Input: `"abc123def456"`
- Output: `"abcdef"`

---

#### `replace`

Replaces all occurrences of a regular expression (`regex`) with a replacement string (`with`). If `recurse = true`, smarterr will apply the replacement until it finds no more matches.

**Example:**

```hcl
transform "replace_numbers" {
  step "replace" {
    regex = "[0-9]+"
    with  = "#"
  }
}
```

- Input: `"abc123def456"`
- Output: `"abc#def#"`

---

#### `trim_space`

Removes leading and trailing whitespace from the value.

**Example:**

```hcl
transform "trim_spaces" {
  step "trim_space" {}
}
```

- Input: `"   hello world   "`
- Output: `"hello world"`

---

#### `fix_space`

Removes leading and trailing whitespace and collapses all internal whitespace sequences to a single space.

**Example:**

```hcl
transform "fix_spaces" {
  step "fix_space" {}
}
```

- Input: `"   hello    world   "`
- Output: `"hello world"`

---

#### `lower`

Converts the value to lowercase.

**Example:**

```hcl
transform "to_lower" {
  step "lower" {}
}
```

- Input: `"HeLLo WoRLD"`
- Output: `"hello world"`

---

#### `upper`

Converts the value to uppercase.

**Example:**

```hcl
transform "to_upper" {
  step "upper" {}
}
```

- Input: `"HeLLo WoRLD"`
- Output: `"HELLO WORLD"`

---

## Notes

- smarterr can layer and merge across directories.
- See [docs/layering.md](layering.md) for details on Config discovery and merging.
- See [docs/diagnostics.md](diagnostics.md) for fallback and diagnostics behavior.
- For advanced stack matching, see the distinction between `call_stack` and `error_stack` sources.
