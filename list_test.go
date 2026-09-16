package smarterr

import (
	"context"
	"errors"
	"testing"

	fwdiag "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
)

// Note: these tests run without SetFS/config, so smarterr uses its fallback
// path. That still produces a diagnostic carrying the original error, which is
// exactly the behavior we need to verify for the list sinks. Config-driven
// enrichment is covered by the AddError/AddEnrich tests.

func collectListResults(seq func(func(list.ListResult) bool)) []list.ListResult {
	var got []list.ListResult
	seq(func(r list.ListResult) bool {
		got = append(got, r)
		return true
	})
	return got
}

func TestNewListResultError_AddsErrorDiagnostic(t *testing.T) {
	ctx := context.Background()
	err := errors.New("listing things failed")

	result := NewListResultError(ctx, err)

	if !result.Diagnostics.HasError() {
		t.Fatalf("expected an error diagnostic, got none")
	}
	if got := len(result.Diagnostics); got != 1 {
		t.Fatalf("expected exactly 1 diagnostic, got %d", got)
	}
	// The fallback path carries the original error text in the detail.
	detail := result.Diagnostics[0].Detail()
	if detail == "" || !contains(detail, "listing things failed") {
		t.Errorf("diagnostic detail = %q, want it to contain the original error", detail)
	}
}

func TestListStreamError_YieldsSingleErrorResult(t *testing.T) {
	ctx := context.Background()
	err := errors.New("decode failure")

	seq := ListStreamError(ctx, err)
	got := collectListResults(seq)

	if len(got) != 1 {
		t.Fatalf("expected exactly 1 streamed result, got %d", len(got))
	}
	if !got[0].Diagnostics.HasError() {
		t.Errorf("streamed result has no error diagnostic")
	}
}

func TestListStreamError_StopsWhenConsumerReturnsFalse(t *testing.T) {
	ctx := context.Background()
	seq := ListStreamError(ctx, errors.New("boom"))

	calls := 0
	seq(func(list.ListResult) bool {
		calls++
		return false // signal stop
	})
	if calls != 1 {
		t.Errorf("expected the push function to be called once, got %d", calls)
	}
}

func TestListStreamEnrich_YieldsEnrichedDiagnostics(t *testing.T) {
	ctx := context.Background()
	var incoming fwdiag.Diagnostics
	incoming.AddError("bad config", "the workspace_id is invalid")

	seq := ListStreamEnrich(ctx, incoming)
	got := collectListResults(seq)

	if len(got) != 1 {
		t.Fatalf("expected exactly 1 streamed result, got %d", len(got))
	}
	if !got[0].Diagnostics.HasError() {
		t.Errorf("enriched result has no error diagnostic")
	}
}

func TestListStreamEnrich_EmptyDiagnostics(t *testing.T) {
	ctx := context.Background()

	seq := ListStreamEnrich(ctx, nil)
	got := collectListResults(seq)

	if len(got) != 1 {
		t.Fatalf("expected exactly 1 streamed result, got %d", len(got))
	}
	if got[0].Diagnostics.HasError() {
		t.Errorf("expected no error diagnostics for empty input, got %d", len(got[0].Diagnostics))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOfSubstr(s, substr) >= 0)
}

func indexOfSubstr(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
