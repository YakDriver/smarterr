package smarterr

import (
	"context"
	"errors"
	"iter"
	"testing"
	"testing/fstest"

	"github.com/hashicorp/terraform-plugin-framework/list"
)

// The "list" happening convention: a stack_match whose called_from targets the
// user's List method and its iterator closures:
//
//	stack_match "list" {
//	  called_from = "\\.List(\\.func[0-9]+)?$"
//	  display     = "listing"
//	}
//
// This regex is anchored so it matches ONLY the user's List method frame
// (`...(*fooListResource).List`) or an anonymous iterator closure defined in it
// (`...List.func1`). It deliberately does NOT match smarterr's own list helpers
// (`ListStreamError`, `NewListResultError`), whose names contain "List" but do
// not end in `.List` or `.List.funcN`. Because enrichment is eager, a
// List-named frame is always live on the stack when these helpers resolve.

const listHappeningHCL = `
template "error_summary" {
  format = "{{.happening}}: {{.error}}"
}
token "happening" {
  source        = "call_stack"
  stack_matches = ["list"]
}
token "error" {
  source = "error"
}
stack_match "list" {
  called_from = "\\.List(\\.func[0-9]+)?$"
  display     = "listing"
}
`

// setListHappeningFS installs an in-memory global config and restores the prior
// package-global FS when the test ends.
func setListHappeningFS(t *testing.T) {
	t.Helper()
	fsys := fstest.MapFS{
		"smarterr/smarterr.hcl": &fstest.MapFile{Data: []byte(listHappeningHCL)},
	}
	prevFS, prevDir := wrappedFS, wrappedBaseDir
	t.Cleanup(func() {
		wrappedFS = prevFS
		wrappedBaseDir = prevDir
	})
	SetFS(&WrappedFS{FS: fsys}, ".")
}

// closureLister mimics a list resource that surfaces a fatal error from inside
// its iterator closure (surface (b)). The closure frame is
// smarterr.closureLister.List.func1.
type closureLister struct{}

func (closureLister) List(ctx context.Context, raise error) iter.Seq[list.ListResult] {
	return func(yield func(list.ListResult) bool) {
		yield(NewListResultError(ctx, raise))
	}
}

// bodyLister mimics a list resource that surfaces a fatal error directly in the
// List method body before streaming (surface (a)). The frame is
// smarterr.bodyLister.List.
type bodyLister struct{}

func (bodyLister) List(ctx context.Context, raise error) iter.Seq[list.ListResult] {
	return ListStreamError(ctx, raise)
}

// TestListHappening_ResolvesFromIteratorClosure is the load-bearing regression
// test for #68: it proves the iterator closure frame survives after List returns
// and that the "list" stack_match resolves the happening from inside the running
// closure. If the closure frame were gone at resolve time, the happening token
// would fall back to empty and the summary would be ": boom".
func TestListHappening_ResolvesFromIteratorClosure(t *testing.T) {
	setListHappeningFS(t)

	// List returns the iterator; the closure has not run yet.
	seq := closureLister{}.List(context.Background(), errors.New("boom"))
	// The "framework" consumes it now, after List has returned.
	results := collectListResults(seq)

	if len(results) != 1 || !results[0].Diagnostics.HasError() {
		t.Fatalf("expected 1 error result, got %#v", results)
	}
	if got, want := results[0].Diagnostics[0].Summary(), "listing: boom"; got != want {
		t.Errorf("summary = %q, want %q\nlist happening did not resolve from the iterator closure frame", got, want)
	}
}

// TestListHappening_ResolvesFromMethodBody covers surface (a): the helper runs
// in the List method body, so the happening resolves against the List method
// frame itself.
func TestListHappening_ResolvesFromMethodBody(t *testing.T) {
	setListHappeningFS(t)

	seq := bodyLister{}.List(context.Background(), errors.New("kaboom"))
	results := collectListResults(seq)

	if len(results) != 1 || !results[0].Diagnostics.HasError() {
		t.Fatalf("expected 1 error result, got %#v", results)
	}
	if got, want := results[0].Diagnostics[0].Summary(), "listing: kaboom"; got != want {
		t.Errorf("summary = %q, want %q", got, want)
	}
}

// TestListHappening_HelperNamesDoNotFalseMatch guards the convention: calling
// the list helpers from a function that is NOT named List must not resolve
// "listing" (otherwise the regex would be matching smarterr's own helper frames
// rather than the user's List method/closure).
func TestListHappening_HelperNamesDoNotFalseMatch(t *testing.T) {
	setListHappeningFS(t)

	// notAList is intentionally not named List.
	notAList := func(ctx context.Context, raise error) iter.Seq[list.ListResult] {
		return ListStreamError(ctx, raise)
	}
	seq := notAList(context.Background(), errors.New("boom"))
	results := collectListResults(seq)

	if len(results) != 1 || !results[0].Diagnostics.HasError() {
		t.Fatalf("expected 1 error result, got %#v", results)
	}
	// happening should be empty here -> summary is ": boom", not "listing: boom".
	if got := results[0].Diagnostics[0].Summary(); got == "listing: boom" {
		t.Errorf("summary = %q; the list regex falsely matched a smarterr helper frame instead of a user List frame", got)
	}
}
