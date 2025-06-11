package internal

import (
	"bytes"
	"testing"
)

func TestEnableDebugAndDebugf(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &Config{SmarterrDebug: &SmarterrDebug{Output: ""}}
	// Patch globalDebugOutput to our buffer for test
	globalDebugOutput = buf
	globalDebugEnabled = false
	Debugf("should not print")
	if buf.Len() != 0 {
		t.Errorf("Expected no debug output when not enabled, got %q", buf.String())
	}
	EnableDebug(cfg)
	Debugf("hello %s", "world")
	if got := buf.String(); got != "[smarterr debug] hello world\n" {
		t.Errorf("Expected debug output, got %q", got)
	}
}
