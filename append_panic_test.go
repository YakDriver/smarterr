// append_panic_test.go
package smarterr

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	sdkdiag "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// panicWalkFS is a FileSystem whose WalkDir panics, driving the panic-recovery
// path inside appendCommon / AppendEnrich (via internal.LoadConfig).
type panicWalkFS struct{}

func (panicWalkFS) Open(string) (fs.File, error)         { return nil, errors.New("unused") }
func (panicWalkFS) ReadFile(string) ([]byte, error)      { return nil, errors.New("unused") }
func (panicWalkFS) Exists(string) bool                   { return true }
func (panicWalkFS) WalkDir(string, fs.WalkDirFunc) error { panic("boom: WalkDir panicked") }

// TestAppend_PanicPreservesDiagnostics verifies that when config loading panics,
// Append/AppendEnrich/AppendOne still return the caller's pre-existing
// diagnostics (plus a fallback/the incoming diagnostics) rather than silently
// discarding everything. Regression test for the unnamed-return bug where the
// deferred recover reassigned a local that never reached the caller.
func TestAppend_PanicPreservesDiagnostics(t *testing.T) {
	prevFS, prevDir := wrappedFS, wrappedBaseDir
	t.Cleanup(func() { wrappedFS, wrappedBaseDir = prevFS, prevDir })
	wrappedFS = panicWalkFS{}
	wrappedBaseDir = "internal"

	pre := sdkdiag.Diagnostics{
		{Severity: sdkdiag.Error, Summary: "pre-existing", Detail: "must survive"},
	}
	ctx := context.Background()

	t.Run("Append", func(t *testing.T) {
		out := Append(ctx, pre, errors.New("real error"))
		if len(out) == 0 {
			t.Fatal("Append returned no diagnostics on panic; caller's diagnostics were lost")
		}
		if out[0].Summary != "pre-existing" {
			t.Errorf("pre-existing diagnostic not preserved; got %+v", out[0])
		}
		var sawPanic bool
		for _, d := range out {
			if strings.Contains(d.Detail, "[smarterr panic:") {
				sawPanic = true
			}
		}
		if !sawPanic {
			t.Errorf("expected a fallback diagnostic noting the panic; got %+v", out)
		}
	})

	t.Run("AppendEnrich", func(t *testing.T) {
		incoming := sdkdiag.Diagnostics{
			{Severity: sdkdiag.Warning, Summary: "incoming", Detail: "from framework"},
		}
		out := AppendEnrich(ctx, pre, incoming)
		if len(out) < 2 {
			t.Fatalf("AppendEnrich lost diagnostics on panic; got %d: %+v", len(out), out)
		}
		if out[0].Summary != "pre-existing" || out[len(out)-1].Summary != "incoming" {
			t.Errorf("expected pre-existing + incoming preserved; got %+v", out)
		}
	})

	t.Run("AppendOne", func(t *testing.T) {
		out := AppendOne(ctx, pre, sdkdiag.Diagnostic{Severity: sdkdiag.Error, Summary: "one", Detail: "single"})
		if len(out) < 2 {
			t.Fatalf("AppendOne lost diagnostics on panic; got %d: %+v", len(out), out)
		}
		if out[0].Summary != "pre-existing" || out[len(out)-1].Summary != "one" {
			t.Errorf("expected pre-existing + single diagnostic preserved; got %+v", out)
		}
	})
}
