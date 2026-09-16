# List resources

Terraform Plugin Framework *list resources* stream results instead of returning
diagnostics. A `List` method sets an `iter.Seq[list.ListResult]` on the stream,
and the framework consumes that iterator later, sometimes on another goroutine.
The resource helpers (`AddError`, `Append`, `AddEnrich`) target a
`diag.Diagnostics`, so they don't cover the list-specific error surfaces on
their own.

smarterr provides three list sinks that reuse the same Config-driven enrichment
as the other helpers.

## Error surfaces and helpers

A typical `List` implementation raises errors in three places:

1. A fatal error before streaming begins, such as a configuration decode
   failure. Use `ListStreamError` (for an `error`) or `ListStreamEnrich` (for a
   framework `diag.Diagnostics`).
1. A fatal per-item or pagination error inside the iterator. Build a result
   with `NewListResultError` and yield it.
1. A per-result, non-fatal error. Append onto the result's own diagnostics with
   `AddError(ctx, &result.Diagnostics, err, ...)`.

```go
func (l *thingListResource) List(ctx context.Context, request list.ListRequest, stream *list.ListResultsStream) {
    // Surface 1: configuration decode.
    var query thingListModel
    if diags := request.Config.Get(ctx, &query); diags.HasError() {
        stream.Results = smarterr.ListStreamEnrich(ctx, diags)
        return
    }

    stream.Results = func(yield func(list.ListResult) bool) {
        for item, err := range listThings(ctx, conn, &input) {
            if err != nil {
                // Surface 2: pagination or item error.
                yield(smarterr.NewListResultError(ctx, err))
                return
            }

            result := request.NewListResult(ctx)
            // Surface 3: per-result error.
            smarterr.AddError(ctx, &result.Diagnostics, flatten(ctx, item, &result))
            if result.Diagnostics.HasError() {
                yield(result)
                return
            }
            if !yield(result) {
                return
            }
        }
    }
}
```

## The "listing" happening convention

smarterr derives the happening word, such as `creating` or `reading`, from the
call stack. List resources need a matching convention. Add a `stack_match` that
targets the `List` method and its iterator closures:

```hcl
stack_match "list" {
  called_from = "\\.List(\\.func[0-9]+)?$"
  display     = "listing"
}

token "happening" {
  source        = "call_stack"
  stack_matches = ["create", "read", "update", "delete", "list"]
}
```

The anchored regex matches the `List` method frame (`...(*fooListResource).List`)
and any anonymous iterator closure defined in it (`...List.func1`). It doesn't
match the smarterr list helpers, whose names contain "List" but don't end in
`.List` or `.List.funcN`.

The order of names in `stack_matches` doesn't set precedence. smarterr walks the
call stack from the innermost frame outward and takes the first frame that
matches any listed rule, so the nearest operation wins. List order affects the
outcome when two rules match the same frame.

### Why this works inside the closure

The list sinks enrich at the moment you call them, while a `List`-named frame
remains live on the stack. They don't defer the work until the framework runs
the iterator:

- `ListStreamError` and `ListStreamEnrich` run in the `List` method body, so the
  `List` method frame drives the match.
- `NewListResultError` runs inside the iterator closure, so the closure frame
  (`List.func1`) drives the match.

A Go iterator closure keeps its parent method name and source file in its
runtime frame. The closure frame stays matchable even after `List` returns and
even when the framework consumes the iterator on another goroutine. The
`smarterr` test suite verifies this behavior for all three surfaces.

## Service and resource enrichment

Context-derived tokens, such as service and resource names, keep working under
`yield`, because they read from `context` rather than the stack. Pass
identifiers as usual:

```go
yield(smarterr.NewListResultError(ctx, err, smarterr.ID, id))
```
