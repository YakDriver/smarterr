// stack_test.go
package internal

import "testing"

// deepFrames recurses `depth` times and then captures the stack, so the live
// stack is at least `depth` frames deep when capture happens.
func deepFrames(depth int) int {
	if depth > 0 {
		return deepFrames(depth - 1)
	}
	return len(CaptureFrames(0))
}

// TestCaptureFrames_NotTruncated proves CaptureFrames grows past its initial
// buffer: a call chain deeper than initialStackDepth must still yield every
// frame rather than being silently truncated.
func TestCaptureFrames_NotTruncated(t *testing.T) {
	depth := initialStackDepth + 16 // force at least one grow-and-retry
	if n := deepFrames(depth); n < depth {
		t.Errorf("CaptureFrames returned %d frames; expected at least the %d nested callers (grow-and-retry did not capture the full stack)", n, depth)
	}
}

// deepGather recurses `depth` times and then calls gatherCallStack, the source
// of the call_stack "happening" token. This is the #77 regression scenario:
// a wrapper chain deeper than the old fixed 10-frame buffer.
func deepGather(depth int) int {
	if depth > 0 {
		return deepGather(depth - 1)
	}
	frames, err := gatherCallStack(3)
	if err != nil {
		return 0
	}
	return len(frames)
}

// TestGatherCallStack_NotTruncated guards against the old fixed 10-frame buffer
// that silently dropped the target frame for deep call chains (#77).
func TestGatherCallStack_NotTruncated(t *testing.T) {
	const depth = 40 // well beyond the old 10-frame cap
	if n := deepGather(depth); n < depth {
		t.Errorf("gatherCallStack captured %d frames through a %d-deep chain; deep stack was truncated", n, depth)
	}
}
