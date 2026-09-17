package smarterr

import (
	"path"
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

// deepStack recurses `remaining` times and then calls captureStack, the origin
// capture behind NewError/Errorf that feeds error_stack tokens (#76).
func deepStack(remaining int) int {
	if remaining > 0 {
		return deepStack(remaining - 1)
	}
	return len(captureStack(0))
}

// TestCaptureStack_NotTruncated guards against the old fixed 16-frame buffer in
// captureStack, which silently dropped the origin frame for deep call chains
// and broke error_stack-sourced tokens (#76).
func TestCaptureStack_NotTruncated(t *testing.T) {
	const depth = 40 // well beyond the old 16-frame cap
	if n := deepStack(depth); n < depth {
		t.Errorf("captureStack captured %d frames through a %d-deep chain; deep stack was truncated", n, depth)
	}
}

// TestCollectRelStackPaths_FindsDeepCaller verifies the full discovery path
// (captureCallers -> frameFiles -> relStackPathsFromFiles) captures a caller
// frame that sits far below the smarterr entry point, guarding against
// reintroducing a fixed-depth stack cap. The wrapper chain lives in
// stack_helper_test.go, so only this test's own frame resolves to
// "stack_test.go"; a truncated capture would keep only the nearby wrapper
// frames and never reach it. Dot mode avoids assuming anything about the
// checkout path.
func TestCollectRelStackPaths_FindsDeepCaller(t *testing.T) {
	const depth = 20 // deeper than any previous fixed cap (5/10/16)
	got := deepWrapper(depth)
	var found bool
	for _, p := range got {
		if path.Base(p) == "stack_test.go" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("collectRelStackPaths did not capture the deep caller frame (stack_test.go) through a %d-deep chain; got %#v", depth, got)
	}
}
