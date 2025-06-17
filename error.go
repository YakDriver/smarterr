package smarterr

import (
	"errors"
	"fmt"
	"runtime"
)

// Error is the enriched smarterr error type.
// It wraps a base error and includes structured annotations
// that can be used by AppendSDK/FW to construct clear, user-friendly diagnostics.
type Error struct {
	Err           error             // The original or wrapped error
	Message       string            // Optional developer-provided message (from Errorf)
	Annotations   map[string]string // Arbitrary key-value annotations (e.g., subaction, resource_id)
	CapturedStack []runtime.Frame   // Captured call stack for stack matching
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Err
}

// Stack returns the captured call stack frames.
func (e *Error) Stack() []runtime.Frame {
	return e.CapturedStack
}

// NewError wraps an existing error with smarterr metadata derived from the call stack.
// It automatically annotates the error with context-aware information (e.g., sub-action)
// without requiring developer input, reducing fragility and promoting consistent error enrichment.
//
// Use NewError at the site where an error is first returned or recognized.
// The resulting error can be passed directly to smarterr.AppendSDK or smarterr.AppendFW
// without needing manual WithField-style annotation.
//
// Example:
//
//	return nil, smarterr.NewError(err)
func NewError(err error) error {
	if err == nil {
		return nil
	}
	stack := captureStack(3) // skip 3 to get the caller of NewError
	return &Error{
		Err:           err,
		Annotations:   map[string]string{},
		CapturedStack: stack,
	}
}

// Errorf formats according to a format specifier and returns a smarterr-enriched error.
// It behaves like fmt.Errorf, but also captures contextual metadata based on the call site.
// This ensures consistent DX and structured diagnostics with minimal developer effort.
//
// Example:
//
//	return smarterr.Errorf("unexpected result for alarm %q", name)
func Errorf(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	stack := captureStack(3) // skip 3 to get the caller of Errorf
	return &Error{
		Err:           errors.New(msg),
		Message:       msg,
		Annotations:   map[string]string{},
		CapturedStack: stack,
	}
}

// Assert wraps a call returning (T, error) with smarterr.NewError on failure.
// Go doesn't yet support generics-based tuple unpacking, so this form works well for now.
func Assert[T any](val T, err error) (T, error) {
	if err != nil {
		return val, NewError(err)
	}
	return val, nil
}
