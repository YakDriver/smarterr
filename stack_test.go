package smarterr

import (
	"context"
	"reflect"
	"testing"
)

func TestRelStackPathsFromFiles(t *testing.T) {
	files := []string{
		"/Users/x/go/src/github.com/hashicorp/terraform-provider-aws/internal/smerr/smarterr.go",
		"/Users/x/go/src/github.com/hashicorp/terraform-provider-aws/internal/service/amp/anomaly_detector_list.go",
		"/Users/x/go/src/github.com/hashicorp/terraform-provider-aws/internal/provider/framework/wrap.go",
		"", // inlined/unknown frame
		"/usr/local/go/src/runtime/proc.go",
	}

	got := relStackPathsFromFiles(files, "internal")
	want := []string{
		"internal/smerr/smarterr.go",
		"internal/service/amp/anomaly_detector_list.go",
		"internal/provider/framework/wrap.go",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("relStackPathsFromFiles() = %#v, want %#v", got, want)
	}

	// The retained baseDir prefix must let config discovery match
	// "<baseDir>/<configDir>" for both the service-specific and shared configs.
	var matchedService bool
	for _, p := range got {
		if p == "internal/service/amp/anomaly_detector_list.go" {
			matchedService = true
		}
	}
	if !matchedService {
		t.Error("expected the AMP service frame to be retained for config discovery")
	}
}

func TestRelStackPathsFromFiles_EmptyBaseDir(t *testing.T) {
	if got := relStackPathsFromFiles([]string{"/a/internal/x.go"}, ""); got != nil {
		t.Errorf("empty baseDir should yield nil, got %#v", got)
	}
}

func TestRelStackPathsFromFiles_NoMatch(t *testing.T) {
	files := []string{"/usr/local/go/src/runtime/proc.go", ""}
	if got := relStackPathsFromFiles(files, "internal"); got != nil {
		t.Errorf("expected nil when no file sits under baseDir, got %#v", got)
	}
}

// TestCaptureCallers_NotTruncated proves the capture is not limited to the old
// fixed depth of 5: a call chain deeper than that must still yield every frame.
// This is the regression guard for the config-discovery truncation bug.
func TestCaptureCallers_NotTruncated(t *testing.T) {
	const depth = 12 // deeper than the previous 5-frame cap
	n := deepCapture(depth)
	if n <= 5 {
		t.Fatalf("captureCallers returned %d frames; expected more than the old cap of 5", n)
	}
	if n < depth {
		t.Errorf("captureCallers returned %d frames; expected at least the %d nested callers", n, depth)
	}
}

// deepCapture recurses `remaining` times and then captures the full stack,
// returning the number of frames captured.
func deepCapture(remaining int) int {
	if remaining > 0 {
		return deepCapture(remaining - 1)
	}
	return len(captureCallers(0))
}

// TestCollectRelStackPaths_FindsDeepCaller verifies that, through several nested
// wrapper frames, collectRelStackPaths still captures this test file's frame.
// The test binary lives under the module path, so baseDir "smarterr" appears in
// the frame's file path.
func TestCollectRelStackPaths_FindsDeepCaller(t *testing.T) {
	got := wrap1()
	var found bool
	for _, p := range got {
		if hasSuffix(p, "stack_test.go") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("collectRelStackPaths did not capture the deep caller frame; got %#v", got)
	}
}

// A short chain of wrappers standing in for smerr shim -> smarterr sink layers.
func wrap1() []string { return wrap2() }
func wrap2() []string { return wrap3() }
func wrap3() []string { return wrap4() }
func wrap4() []string { return collectRelStackPaths(context.Background(), "smarterr") }

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
