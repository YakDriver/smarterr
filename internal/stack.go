// stack.go
// Shared, truncation-safe call-stack capture for smarterr.
package internal

import "runtime"

// initialStackDepth is the starting buffer size for stack capture. It's large
// enough to hold the vast majority of real call chains in a single pass; deeper
// chains simply grow the buffer (see CaptureCallers).
const initialStackDepth = 64

// CaptureCallers captures the entire call stack as program counters, growing the
// buffer until it fits. Unlike a fixed-size buffer, this never silently
// truncates deep call chains: runtime.Callers only fills as many slots as the
// buffer holds and signals nothing beyond the returned count, so a fixed buffer
// loses frames above its size. Config discovery and stack_match resolution walk
// these frames, so a lost frame means a silently missed match.
//
// skip is passed straight to runtime.Callers (0 = runtime.Callers itself,
// 1 = CaptureCallers, 2 = CaptureCallers' caller). A wrapper that delegates to
// CaptureCallers should therefore pass its own skip + 1 to account for the
// extra frame CaptureCallers adds.
//
// Errors are a cold path, so capturing the full stack is inexpensive.
func CaptureCallers(skip int) []uintptr {
	pcs := make([]uintptr, initialStackDepth)
	for {
		n := runtime.Callers(skip, pcs)
		if n < len(pcs) {
			return pcs[:n]
		}
		// Buffer was completely filled; more frames may exist above it. Grow
		// and retry until the whole stack fits.
		pcs = make([]uintptr, 2*len(pcs))
	}
}

// CaptureFrames captures the entire call stack as resolved runtime.Frames,
// growing the buffer until it fits (see CaptureCallers for the rationale).
//
// skip follows the same convention as CaptureCallers: it is passed straight to
// runtime.Callers, so a wrapper delegating to CaptureFrames should pass its own
// skip + 1. Returns nil if no frames are available.
func CaptureFrames(skip int) []runtime.Frame {
	pcs := make([]uintptr, initialStackDepth)
	var n int
	for {
		n = runtime.Callers(skip, pcs)
		if n < len(pcs) {
			break
		}
		pcs = make([]uintptr, 2*len(pcs))
	}
	if n == 0 {
		return nil
	}
	frames := runtime.CallersFrames(pcs[:n])
	var result []runtime.Frame
	for {
		frame, more := frames.Next()
		result = append(result, frame)
		if !more {
			break
		}
	}
	return result
}
