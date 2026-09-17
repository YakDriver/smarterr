// transform_strip_test.go
package internal

import (
	"testing"
	"time"
)

// TestApplyStrip_EmptyValueNoHang verifies that an empty (present-but-"")
// strip_prefix/strip_suffix Value is a no-op and does not hang, including with
// recurse=true. Regression test for the infinite loop where
// strings.CutPrefix/CutSuffix always report a match against an empty separator.
func TestApplyStrip_EmptyValueNoHang(t *testing.T) {
	empty := ""
	recurse := true

	cases := []struct {
		name string
		fn   func(string, TransformStep) string
		step TransformStep
		in   string
		want string
	}{
		{
			name: "strip_prefix empty value recurse",
			fn:   applyStripPrefix,
			step: TransformStep{Value: &empty, Recurse: &recurse},
			in:   "  some error message  ",
			want: "some error message",
		},
		{
			name: "strip_suffix empty value recurse",
			fn:   applyStripSuffix,
			step: TransformStep{Value: &empty, Recurse: &recurse},
			in:   "some error message,",
			want: "some error message,",
		},
		{
			name: "strip_prefix empty value no recurse",
			fn:   applyStripPrefix,
			step: TransformStep{Value: &empty},
			in:   "  keep me  ",
			want: "keep me",
		},
		{
			name: "strip_suffix empty value no recurse",
			fn:   applyStripSuffix,
			step: TransformStep{Value: &empty},
			in:   "  keep me  ",
			want: "keep me",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			done := make(chan string, 1)
			go func() { done <- tc.fn(tc.in, tc.step) }()
			select {
			case got := <-done:
				if got != tc.want {
					t.Errorf("got %q, want %q", got, tc.want)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("timed out: empty Value caused an infinite loop")
			}
		})
	}
}

// TestApplyStrip_NonEmptyValueRecurse confirms recurse still strips repeated
// non-empty prefixes/suffixes (guards against the fix over-broadening).
func TestApplyStrip_NonEmptyValueRecurse(t *testing.T) {
	pre := "ab"
	suf := ","
	recurse := true

	if got := applyStripPrefix("ababkeep", TransformStep{Value: &pre, Recurse: &recurse}); got != "keep" {
		t.Errorf("strip_prefix recurse got %q, want %q", got, "keep")
	}
	if got := applyStripSuffix("keep,,,", TransformStep{Value: &suf, Recurse: &recurse}); got != "keep" {
		t.Errorf("strip_suffix recurse got %q, want %q", got, "keep")
	}
}
