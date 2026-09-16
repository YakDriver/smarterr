package smarterr

import (
	"errors"
	"fmt"
	"testing"
)

// sentinel is a typed error used to verify errors.As traversal.
type sentinel struct{ msg string }

func (s *sentinel) Error() string { return s.msg }

func TestErrorf_WrapVerb_ErrorsIs(t *testing.T) {
	base := errors.New("base failure")

	err := Errorf("operation failed: %w", base)

	if !errors.Is(err, base) {
		t.Fatalf("errors.Is(err, base) = false, want true; %%w chain not preserved")
	}

	want := "operation failed: base failure"
	if got := err.Error(); got != want {
		t.Errorf("err.Error() = %q, want %q", got, want)
	}
}

func TestErrorf_WrapVerb_ErrorsAs(t *testing.T) {
	target := &sentinel{msg: "typed failure"}

	err := Errorf("wrapped: %w", target)

	var got *sentinel
	if !errors.As(err, &got) {
		t.Fatalf("errors.As(err, &got) = false, want true; typed error not reachable through chain")
	}
	if got.msg != target.msg {
		t.Errorf("errors.As recovered msg = %q, want %q", got.msg, target.msg)
	}
}

func TestErrorf_WrapVerb_Unwrap(t *testing.T) {
	base := errors.New("root cause")

	err := Errorf("context: %w", base)

	// The smarterr.Error unwraps to the fmt-wrapped error, which in turn
	// unwraps to the base error. errors.Is walks the whole chain.
	if unwrapped := errors.Unwrap(err); unwrapped == nil {
		t.Fatal("errors.Unwrap(err) = nil, want non-nil (chain broken)")
	}
	if !errors.Is(errors.Unwrap(err), base) {
		t.Errorf("unwrapped error does not lead to base")
	}
}

func TestErrorf_MultipleWrapVerbs(t *testing.T) {
	err1 := errors.New("first")
	err2 := errors.New("second")

	err := Errorf("multi: %w and %w", err1, err2)

	if !errors.Is(err, err1) {
		t.Errorf("errors.Is(err, err1) = false, want true")
	}
	if !errors.Is(err, err2) {
		t.Errorf("errors.Is(err, err2) = false, want true")
	}
}

func TestErrorf_NoWrapVerb(t *testing.T) {
	err := Errorf("plain %s message", "formatted")

	want := "plain formatted message"
	if got := err.Error(); got != want {
		t.Errorf("err.Error() = %q, want %q", got, want)
	}

	// A non-wrapping Errorf must not spuriously match an unrelated error.
	if errors.Is(err, errors.New("plain formatted message")) {
		t.Errorf("errors.Is matched an unrelated error by message")
	}
}

func TestErrorf_CapturesStack(t *testing.T) {
	err := Errorf("boom: %w", errors.New("inner"))

	var se *Error
	if !errors.As(err, &se) {
		t.Fatalf("errors.As(err, *Error) = false, want true")
	}
	if len(se.Stack()) == 0 {
		t.Errorf("Errorf did not capture a call stack")
	}
}

func TestNewError_PreservesChain(t *testing.T) {
	base := errors.New("origin")
	wrapped := fmt.Errorf("layer: %w", base)

	err := NewError(wrapped)

	if !errors.Is(err, base) {
		t.Errorf("errors.Is(NewError(wrapped), base) = false, want true")
	}
}
