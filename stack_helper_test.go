// stack_helper_test.go
package smarterr

import "context"

// deepWrapper recurses `depth` times and then invokes collectRelStackPaths in
// dot mode (which passes frame paths through unchanged). Every frame in this
// chain lives in stack_helper_test.go, so the only captured frame that resolves
// to "stack_test.go" is the original test caller — which sits deeper than every
// wrapper frame. A truncated capture would retain only the nearby wrapper frames
// and never reach the test caller, so it could not satisfy a "stack_test.go"
// assertion. This keeps TestCollectRelStackPaths_FindsDeepCaller a real
// truncation guard, and dot mode avoids assuming the checkout path contains any
// particular directory segment.
func deepWrapper(depth int) []string {
	if depth > 0 {
		return deepWrapper(depth - 1)
	}
	return collectRelStackPaths(context.Background(), ".")
}
