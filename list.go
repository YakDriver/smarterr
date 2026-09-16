package smarterr

import (
	"context"
	"iter"

	fwdiag "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
)

// This file provides smarterr sinks for Terraform Plugin Framework *list
// resources*. Unlike CRUD methods, a ListResource.List method does not return
// a diag.Diagnostics. Instead it streams results by setting an
// iter.Seq[list.ListResult] on the stream, and surfaces errors as ListResult
// values whose Diagnostics field is populated.
//
// There are three error surfaces in a typical List implementation:
//
//  1. A fatal error before streaming (e.g., config decode). Mirror
//     list.ListResultsStreamDiagnostics with ListStreamError / ListStreamEnrich.
//  2. A fatal per-item or pagination error inside the iterator. Build a
//     ListResult with NewListResultError and yield it.
//  3. A per-result (non-fatal) error. Append onto the result's own
//     diagnostics with AddError(ctx, &result.Diagnostics, err, ...).
//
// All three reuse the same Config-driven enrichment as AddError/AddEnrich, so
// list resources get the same benefits as CRUD call sites with a single call.
//
// Enrichment is eager: these helpers format the diagnostic at call time, while
// the List method (or its iterator closure) is still on the call stack. That
// keeps stack-derived tokens (config layering and "happening" detection)
// working even though the framework consumes the returned iterator lazily,
// possibly on another goroutine.

// NewListResultError builds a list.ListResult carrying a single
// smarterr-enriched error diagnostic. Use it inside a ListResource.List
// iterator to surface a fatal per-item or pagination error with the same
// Config-driven formatting as AddError.
//
// Example:
//
//	for item, err := range listThings(ctx, conn, &input) {
//	    if err != nil {
//	        yield(smarterr.NewListResultError(ctx, err))
//	        return
//	    }
//	    // ...
//	}
func NewListResultError(ctx context.Context, err error, keyvals ...any) list.ListResult {
	var result list.ListResult
	AddError(ctx, &result.Diagnostics, err, keyvals...)
	return result
}

// ListStreamError returns an iter.Seq[list.ListResult] that yields exactly one
// result containing a smarterr-enriched error diagnostic. Use it for a fatal
// error that occurs before streaming begins, mirroring
// list.ListResultsStreamDiagnostics but with enrichment.
//
// Example:
//
//	out, err := findThings(ctx, conn)
//	if err != nil {
//	    stream.Results = smarterr.ListStreamError(ctx, err)
//	    return
//	}
func ListStreamError(ctx context.Context, err error, keyvals ...any) iter.Seq[list.ListResult] {
	result := NewListResultError(ctx, err, keyvals...)
	return func(yield func(list.ListResult) bool) {
		yield(result)
	}
}

// ListStreamEnrich returns an iter.Seq[list.ListResult] that yields exactly one
// result whose diagnostics are the smarterr-enriched form of the incoming
// framework diagnostics. Use it for the config-decode path, where the framework
// hands back a diag.Diagnostics rather than an error.
//
// Example:
//
//	if diags := request.Config.Get(ctx, &query); diags.HasError() {
//	    stream.Results = smarterr.ListStreamEnrich(ctx, diags)
//	    return
//	}
func ListStreamEnrich(ctx context.Context, incoming fwdiag.Diagnostics, keyvals ...any) iter.Seq[list.ListResult] {
	var enriched fwdiag.Diagnostics
	AddEnrich(ctx, &enriched, incoming, keyvals...)
	return func(yield func(list.ListResult) bool) {
		yield(list.ListResult{Diagnostics: enriched})
	}
}
