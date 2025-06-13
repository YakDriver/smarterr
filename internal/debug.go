// debug.go
// Internal debug output for smarterr itself.
//
// This file implements a simple on/off debug logger for smarterr's own diagnostics.
// If the smarterr_debug block is present in config, debug output is enabled; otherwise, it is off.
// User-facing logging is handled in the root package (see logger.go).

package internal

import (
	"fmt"
	"io"
	"os"
	"sync"
)

var (
	// globalDebugEnabled controls whether internal debug output is enabled.
	globalDebugEnabled bool
	// globalDebugOutput is the writer for internal debug output.
	globalDebugOutput io.Writer = os.Stderr
	debugMutex        sync.Mutex
)

// EnableDebug sets up internal debug output based on the Smarterr block in config.
func EnableDebug(cfg *Config) {
	debugMutex.Lock()
	defer debugMutex.Unlock()
	if cfg != nil && cfg.Smarterr != nil && cfg.Smarterr.Debug {
		globalDebugEnabled = true
		// Always use stderr for now; can extend later if needed
		globalDebugOutput = os.Stderr
	} else {
		globalDebugEnabled = false
		globalDebugOutput = os.Stderr
	}
}

// Debugf emits a debug message if internal debug output is enabled.
func Debugf(format string, args ...any) {
	debugMutex.Lock()
	enabled := globalDebugEnabled
	out := globalDebugOutput
	debugMutex.Unlock()
	if enabled {
		fmt.Fprintf(out, "[smarterr debug] "+format+"\n", args...)
	}
}
