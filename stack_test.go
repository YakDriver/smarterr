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

// baseDir "." is a supported mode (embed root == working dir). Frame files are
// absolute and can't be anchored on "./", so non-empty paths pass through so
// candidate matching (bare configDir) can still find them.
// Windows frame paths use backslashes, while io/fs config paths use "/". The
// extractor must normalize so discovery works on Windows too.
func TestRelStackPathsFromFiles_WindowsPaths(t *testing.T) {
	files := []string{
		`C:\repo\internal\service\amp\anomaly_detector_list.go`,
		`C:\deps\notinternal\service\amp\x.go`, // substring, not a segment: ignore
		`internal\service\acm\certificate.go`,  // baseDir at start
	}
	got := relStackPathsFromFiles(files, "internal")
	want := []string{
		"internal/service/amp/anomaly_detector_list.go",
		"internal/service/acm/certificate.go",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("relStackPathsFromFiles(windows) = %#v, want %#v", got, want)
	}
}

func TestRelStackPathsFromFiles_DotBaseDir(t *testing.T) {
	files := []string{
		"/abs/proj/service/amp/anomaly_detector_list.go",
		"",
		"/usr/local/go/src/runtime/proc.go",
	}
	got := relStackPathsFromFiles(files, ".")
	want := []string{
		"/abs/proj/service/amp/anomaly_detector_list.go",
		"/usr/local/go/src/runtime/proc.go",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("relStackPathsFromFiles(., ...) = %#v, want %#v", got, want)
	}
}

func TestRelStackPathsFromFiles_NoMatch(t *testing.T) {
	files := []string{"/usr/local/go/src/runtime/proc.go", ""}
	if got := relStackPathsFromFiles(files, "internal"); got != nil {
		t.Errorf("expected nil when no file sits under baseDir, got %#v", got)
	}
}

// baseDir must match only at a path-segment boundary. A directory that merely
// contains baseDir as a substring (e.g. "notinternal") must not be treated as a
// frame under the configured root; a genuine "/internal/" segment must be.
func TestRelStackPathsFromFiles_SegmentBoundary(t *testing.T) {
	files := []string{
		"/deps/notinternal/service/amp/x.go",          // substring, not a segment: ignore
		"/deps/notinternal/internal/service/amp/y.go", // real "/internal/" segment: keep from there
		"internal/service/amp/z.go",                   // baseDir at start: keep as-is
	}
	got := relStackPathsFromFiles(files, "internal")
	want := []string{
		"internal/service/amp/y.go",
		"internal/service/amp/z.go",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("relStackPathsFromFiles() = %#v, want %#v", got, want)
	}
}

// TestCaptureCallers_NotTruncated proves the capture grows past its initial
// buffer: a call chain deeper than the 64-frame starting size must still yield
// every frame. This exercises the grow-and-retry branch (and guards against the
// original 5-frame truncation).
func TestCaptureCallers_NotTruncated(t *testing.T) {
	const depth = 80 // exceeds captureCallers' initial 64-slot buffer
	n := deepCapture(depth)
	if n < depth {
		t.Errorf("captureCallers returned %d frames; expected at least the %d nested callers (grow-and-retry did not capture the full stack)", n, depth)
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
