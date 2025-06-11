package internal

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestToken_Resolve(t *testing.T) {
	tests := []struct {
		name     string
		token    Token
		ctx      context.Context
		runtime  *Runtime
		want     string
		hasError bool
	}{
		// Parameter source tests
		{
			name: "Parameter source - valid parameter",
			token: Token{
				Name:      "example",
				Source:    "parameter",
				Parameter: stringPtr("param1"),
			},
			ctx: context.Background(),
			runtime: NewRuntime(&Config{
				Parameters: []Parameter{
					{Name: "param1", Value: "value1"},
				},
			}, nil, nil),
			want:     "value1",
			hasError: false,
		},
		{
			name: "Parameter source - parameter not found",
			token: Token{
				Name:      "example",
				Source:    "parameter",
				Parameter: stringPtr("param2"),
			},
			ctx: context.Background(),
			runtime: NewRuntime(&Config{
				Parameters: []Parameter{
					{Name: "param1", Value: "value1"},
				},
			}, nil, nil),
			want:     "",
			hasError: false, // No error is returned, but fallback is used
		},

		// Context source tests
		{
			name: "Context source - valid context value",
			token: Token{
				Name:    "example",
				Source:  "context",
				Context: stringPtr("key1"),
			},
			ctx:      context.WithValue(context.Background(), "key1", "value1"),
			runtime:  NewRuntime(&Config{}, nil, nil),
			want:     "value1",
			hasError: false,
		},
		{
			name: "Context source - context key not found",
			token: Token{
				Name:    "example",
				Source:  "context",
				Context: stringPtr("key2"),
			},
			ctx:      context.WithValue(context.Background(), "key1", "value1"),
			runtime:  NewRuntime(&Config{}, nil, nil),
			want:     "",
			hasError: false, // No error is returned, but fallback is used
		},

		// Error source tests
		{
			name: "Error source - valid error field",
			token: Token{
				Name:   "example",
				Source: "error",
			},
			ctx:      context.Background(),
			runtime:  NewRuntime(&Config{TokenErrorMode: "placeholder"}, errors.New("example error"), nil),
			want:     "<example>",
			hasError: false,
		},
		{
			name: "Error source - no error in runtime",
			token: Token{
				Name:   "example",
				Source: "error",
			},
			ctx:      context.Background(),
			runtime:  NewRuntime(&Config{}, nil, nil),
			want:     "",
			hasError: false, // No error is returned, but fallback is used
		},
		{
			name: "Error source - valid error field (detailed mode)",
			token: Token{
				Name:   "example",
				Source: "error",
			},
			ctx:      context.Background(),
			runtime:  NewRuntime(&Config{TokenErrorMode: "detailed"}, errors.New("example error"), nil),
			want:     "[unresolved token: example]",
			hasError: false,
		},
		{
			name: "Error source - valid error field (empty mode)",
			token: Token{
				Name:   "example",
				Source: "error",
			},
			ctx:      context.Background(),
			runtime:  NewRuntime(&Config{TokenErrorMode: "empty"}, errors.New("example error"), nil),
			want:     "",
			hasError: false,
		},

		// Arg source tests
		{
			name: "Arg source - valid argument",
			token: Token{
				Name:   "arg1",
				Source: "arg",
				Arg:    stringPtr("arg1"),
			},
			ctx:      context.Background(),
			runtime:  NewRuntime(&Config{}, nil, nil, "arg1", "value1"),
			want:     "value1",
			hasError: false,
		},
		{
			name: "Arg source - argument not found",
			token: Token{
				Name:   "arg2",
				Source: "arg",
				Arg:    stringPtr("arg2"),
			},
			ctx:      context.Background(),
			runtime:  NewRuntime(&Config{}, nil, nil, "arg1", "value1"),
			want:     "",
			hasError: false, // No error is returned, but fallback is used
		},

		// Default case tests
		{
			name: "Unknown source",
			token: Token{
				Name:   "example",
				Source: "unknown",
			},
			ctx:      context.Background(),
			runtime:  NewRuntime(&Config{}, nil, nil),
			want:     "",
			hasError: false, // No error is returned, but fallback is used
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.Resolve(tt.ctx, tt.runtime)
			if got != tt.want {
				t.Errorf("Resolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessStackMatches(t *testing.T) {
	tests := []struct {
		name          string
		stackMatches  []StackMatch
		frames        []runtime.Frame
		expected      string
		expectedError bool
	}{
		{
			name: "Match update function",
			stackMatches: []StackMatch{
				{Name: "update", CalledFrom: "resource[a-zA-Z0-9]*Update", Display: "updating"},
			},
			frames: []runtime.Frame{
				{Function: "resourceExampleUpdate"},
			},
			expected:      "updating",
			expectedError: false,
		},
		{
			name: "Match read with Set called before",
			stackMatches: []StackMatch{
				{Name: "read_set", CalledFrom: "resource[a-zA-Z0-9]*Read", CalledAfter: "Set", Display: "setting during read"},
			},
			frames: []runtime.Frame{
				{Function: "Set"},
				{Function: "resourceExampleRead"},
			},
			expected:      "setting during read",
			expectedError: false,
		},
		{
			name: "Match read with find called before",
			stackMatches: []StackMatch{
				{Name: "read_find", CalledFrom: "resource[a-zA-Z0-9]*Read", CalledAfter: "find.*", Display: "finding during read"},
			},
			frames: []runtime.Frame{
				{Function: "findResource"},
				{Function: "resourceExampleRead"},
			},
			expected:      "finding during read",
			expectedError: false,
		},
		{
			name: "Match create with wait called before",
			stackMatches: []StackMatch{
				{Name: "create_wait", CalledFrom: "resource[a-zA-Z0-9]*Create", CalledAfter: "wait.*", Display: "waiting during creation"},
			},
			frames: []runtime.Frame{
				{Function: "waitForResource"},
				{Function: "resourceExampleCreate"},
			},
			expected:      "waiting during creation",
			expectedError: false,
		},
		{
			name: "No match for update function",
			stackMatches: []StackMatch{
				{Name: "update", CalledFrom: "resource[a-zA-Z0-9]*Update", Display: "updating"},
			},
			frames: []runtime.Frame{
				{Function: "resourceExampleRead"},
			},
			expected:      "",
			expectedError: false,
		},
		{
			name: "Invalid regex in CalledFrom",
			stackMatches: []StackMatch{
				{Name: "invalid_regex", CalledFrom: "[invalid", Display: "invalid regex"},
			},
			frames: []runtime.Frame{
				{Function: "resourceExampleUpdate"},
			},
			expected:      "",
			expectedError: true,
		},
		{
			name: "Invalid regex in CalledAfter",
			stackMatches: []StackMatch{
				{Name: "invalid_regex", CalledFrom: "resource[a-zA-Z0-9]*Read", CalledAfter: "[invalid", Display: "invalid regex"},
			},
			frames: []runtime.Frame{
				{Function: "Set"},
				{Function: "resourceExampleRead"},
			},
			expected:      "",
			expectedError: true,
		},
		{
			name: "No stack frames",
			stackMatches: []StackMatch{
				{Name: "update", CalledFrom: "resource[a-zA-Z0-9]*Update", Display: "updating"},
			},
			frames:        []runtime.Frame{},
			expected:      "",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processStackMatches(tt.stackMatches, tt.frames)
			if (err != nil) != tt.expectedError {
				t.Errorf("processStackMatches() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if result != tt.expected {
				t.Errorf("processStackMatches() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// It's tough to test the actual call stack, but we can at least check that
// the function returns a non-empty slice of frames and that they have the
// expected structure.
func TestGatherCallStackFrameStructure(t *testing.T) {
	frames, err := gatherCallStack(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, frame := range frames {
		if frame.Function == "" {
			t.Errorf("expected non-empty function name, got empty")
		}
		if frame.File == "" {
			t.Errorf("expected non-empty file path, got empty")
		}
	}
}

// It's tough to test the actual call stack, but we can at least check that
// the test function is present in the call stack.
func TestGatherCallStack(t *testing.T) {
	// Call gatherCallStack with a known skip value
	frames, err := gatherCallStack(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ensure that the returned frames are not empty
	if len(frames) == 0 {
		t.Fatalf("expected non-empty call stack, got empty")
	}

	// Check that the first frame is this test function
	found := false
	for _, frame := range frames {
		if strings.HasSuffix(frame.Function, "TestGatherCallStack") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find TestGatherCallStack in call stack, but it was not found")
	}
}

func TestParseKeyvals(t *testing.T) {
	tests := []struct {
		name  string
		input []any
		want  map[string]any
	}{
		{
			name:  "Valid key-value pairs",
			input: []any{"id", "rds", "service", "Provider"},
			want:  map[string]any{"id": "rds", "service": "Provider"},
		},
		{
			name:  "Odd number of arguments",
			input: []any{"id", "rds", "service"},
			want:  map[string]any{"id": "rds"},
		},
		{
			name:  "Non-string key",
			input: []any{123, "rds", "service", "Provider"},
			want:  map[string]any{},
		},
		{
			name:  "Empty input",
			input: []any{},
			want:  map[string]any{},
		},
		{
			name:  "Duplicate keys",
			input: []any{"id", "rds", "id", "new_rds"},
			want:  map[string]any{"id": "new_rds"},
		},
		{
			name:  "Nil value",
			input: []any{"id", nil, "service", "Provider"},
			want:  map[string]any{"id": nil, "service": "Provider"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call parseKeyvals
			got := parseKeyvals(tt.input...)

			// Verify the result
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseKeyvals() = %v, want %v", got, tt.want)
			}
		})
	}
}
