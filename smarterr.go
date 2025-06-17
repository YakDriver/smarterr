package smarterr

import (
	"context"
	"runtime"

	"github.com/YakDriver/smarterr/internal"
	fwdiag "github.com/hashicorp/terraform-plugin-framework/diag"
	sdkdiag "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// Re-export internal.Debugf for internal debugging
var Debugf = internal.Debugf

var (
	wrappedFS      FileSystem
	wrappedBaseDir string
)

// SetFS allows the host application to provide a FileSystem implementation and the base directory for path normalization.
func SetFS(fs FileSystem, baseDir string) {
	Debugf("SetFS called with baseDir=%q", baseDir)
	wrappedFS = fs
	wrappedBaseDir = baseDir
}

// EnrichAppendFW is a helper function that enriches diagnostics with smarterr information.
// This will not change the severity of either incoming or existing diagnostics, but will change
// the summary and detail of _incoming_ diagnostics only with smarterr information.
func EnrichAppendFW(ctx context.Context, existing fwdiag.Diagnostics, incoming fwdiag.Diagnostics, keyvals ...any) {
	for _, diag := range incoming {
		if diag == nil {
			continue
		}

		// Need care here to deduplicate if either the incoming unenriched diagnostics
		// or the incoming enriched diagnostics already exist in the existing diagnostics.

		if existing.Contains(diag) {
			continue
		}
		existing = append(existing, diag)
	}
}

func AppendFW(ctx context.Context, diags fwdiag.Diagnostics, err error, keyvals ...any) {
	Debugf("AppendFW called with error: %v", err)
	defer func() {
		if r := recover(); r != nil {
			Debugf("Panic recovered in AppendFW: %v", r)
			// Fallback: original error summary, panic at end of detail
			summary := firstNWords(err, 3)
			detail := ""
			if err != nil {
				detail = err.Error()
			}
			panicMsg := " [smarterr panic: "
			switch v := r.(type) {
			case error:
				panicMsg += v.Error()
			case string:
				panicMsg += v
			default:
				panicMsg += "unknown panic"
			}
			panicMsg += "]"
			detail += panicMsg
			diags.AddError(summary, detail)
		}
	}()
	appendCommon(ctx, func(summary, detail string) {
		Debugf("AppendFW add error: summary=%q detail=%q", summary, detail)
		diags.AddError(summary, detail)
	}, err, keyvals...)
}

func AppendSDK(ctx context.Context, diags sdkdiag.Diagnostics, err error, keyvals ...any) sdkdiag.Diagnostics {
	Debugf("AppendSDK called with error: %v", err)
	defer func() {
		if r := recover(); r != nil {
			Debugf("Panic recovered in AppendSDK: %v", r)
			// Fallback: original error summary, panic at end of detail
			summary := firstNWords(err, 3)
			detail := ""
			if err != nil {
				detail = err.Error()
			}
			panicMsg := " [smarterr panic: "
			switch v := r.(type) {
			case error:
				panicMsg += v.Error()
			case string:
				panicMsg += v
			default:
				panicMsg += "unknown panic"
			}
			panicMsg += "]"
			detail += panicMsg
			diags = append(diags, sdkdiag.Diagnostic{
				Severity: sdkdiag.Error,
				Summary:  summary,
				Detail:   detail,
			})
		}
	}()
	appendCommon(ctx, func(summary, detail string) {
		Debugf("AppendSDK add error: summary=%q detail=%q", summary, detail)
		diags = append(diags, sdkdiag.Diagnostic{
			Severity: sdkdiag.Error,
			Summary:  summary,
			Detail:   detail,
		})
	}, err, keyvals...)
	return diags
}

// appendCommon is a shared helper for AppendFW and AppendSDK that resolves and formats error messages
// using the smarterr configuration. It attempts to load configuration from the embedded filesystem and
// the caller's directory, then builds a runtime to render the final error message. If any step fails,
// it appends a fallback error message that always includes the original error (if present) in the summary.
// The add function is used to append the error to the diagnostics in a way appropriate for the caller.
func appendCommon(ctx context.Context, add func(summary, detail string), err error, keyvals ...any) {
	Debugf("appendCommon called with error: %v, keyvals: %v", err, keyvals)
	var diagnostics []error
	if wrappedFS == nil {
		Debugf("No wrappedFS set; calling addFallbackInitError")
		addFallbackInitError(add, err)
		return
	}

	relStackPaths := collectRelStackPaths(wrappedBaseDir)
	Debugf("collectRelStackPaths returned: %v", relStackPaths)
	cfg, cfgErr := internal.LoadConfigWithDiagnostics(wrappedFS, relStackPaths, wrappedBaseDir, &diagnostics)
	if cfgErr != nil {
		Debugf("Config load error: %v", cfgErr)
		addFallbackConfigError(add, err, cfgErr)
		return
	}

	rt := internal.NewRuntimeWithDiagnostics(cfg, err, nil, &diagnostics, keyvals...)
	values := rt.BuildTokenValueMap(ctx)

	summary, detail := renderDiagnosticsWithDiagnostics(cfg, err, values, &diagnostics)
	Debugf("renderDiagnostics returned summary=%q detail=%q", summary, detail)
	add(summary, detail)
	emitLogTemplates(ctx, cfg, values)
}

// captureStack returns a slice of runtime.Frames for the current call stack, skipping 'skip' frames.
func captureStack(skip int) []runtime.Frame {
	pcs := make([]uintptr, 16)
	n := runtime.Callers(skip, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	var stack []runtime.Frame
	for {
		frame, more := frames.Next()
		stack = append(stack, frame)
		if !more {
			break
		}
	}
	return stack
}

// addFallbackInitError handles the fallback for missing FS.
func addFallbackInitError(add func(summary, detail string), err error) {
	Debugf("addFallbackInitError called with error: %v", err)
	summary := firstNWords(err, 3)
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	detail += " [smarterr initialization: Embedded filesystem not set, use SetFS()]"
	add(summary, detail)
}

// addFallbackConfigError handles the fallback for config load errors.
func addFallbackConfigError(add func(summary, detail string), err error, cfgErr error) {
	Debugf("addFallbackConfigError called with error: %v, cfgErr: %v", err, cfgErr)
	summary := firstNWords(err, 3)
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	detail += " [smarterr Configuration Error: " + cfgErr.Error() + "]"
	add(summary, detail)
}

// collectRelStackPaths normalizes call stack file paths relative to wrappedBaseDir.
func collectRelStackPaths(baseDir string) []string {
	Debugf("collectRelStackPaths called with baseDir=%q", baseDir)
	const stackDepth = 5
	pcs := make([]uintptr, stackDepth)
	n := runtime.Callers(2, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	var relStackPaths []string
	for i := range n {
		frame, more := frames.Next()
		if frame.File != "" && baseDir != "" {
			idx := indexOf(frame.File, baseDir+"/")
			if idx != -1 {
				rel := frame.File[idx+len(baseDir)+1:]
				Debugf("Stack frame %d: file=%q rel=%q", i, frame.File, rel)
				relStackPaths = append(relStackPaths, rel)
			}
		}
		if !more {
			break
		}
	}
	return relStackPaths
}

// renderDiagnostics renders summary and detail, with fallback if templates fail.
func renderDiagnosticsWithDiagnostics(cfg *internal.Config, err error, values map[string]any, diagnostics *[]error) (string, string) {
	Debugf("renderDiagnostics called with error: %v, values: %v", err, values)
	summaryTmpl, summaryErr := cfg.RenderTemplate("error_summary", values)
	var summary string
	if summaryErr != nil {
		Debugf("Summary template error: %v", summaryErr)
		*diagnostics = append(*diagnostics, summaryErr)
		summary = firstNWords(err, 3)
	} else {
		summary = summaryTmpl
	}
	detailTmpl, detailErr := cfg.RenderTemplate("error_detail", values)
	var detail string
	if detailErr != nil || summaryErr != nil {
		Debugf("Detail template error: %v", detailErr)
		if detailErr != nil {
			*diagnostics = append(*diagnostics, detailErr)
		}
		detail = ""
		if err != nil {
			detail = err.Error()
		}
		problems := ""
		if summaryErr != nil {
			problems += " [smarterr summary template error: " + summaryErr.Error() + "]"
		}
		if detailErr != nil {
			problems += " [smarterr detail template error: " + detailErr.Error() + "]"
		}
		detail += problems
	} else {
		detail = detailTmpl
	}
	// Append diagnostics if present
	if diagnostics != nil && len(*diagnostics) > 0 {
		detail += "\n\n[smarterr diagnostics]"
		for _, diag := range *diagnostics {
			Debugf("[smarterr diagnostics] %v", diag)
			detail += "\n- " + diag.Error()
		}
	}
	return summary, detail
}

// emitLogTemplates checks for log_error, log_warn, and log_info templates and emits logs if present.
func emitLogTemplates(ctx context.Context, cfg *internal.Config, values map[string]any) {
	Debugf("emitLogTemplates called")
	if globalLogger == nil {
		Debugf("No globalLogger set; skipping user-facing log emission")
		return
	}
	if tmpl, err := cfg.RenderTemplate("log_error", values); err == nil && tmpl != "" {
		Debugf("Emitting user-facing log_error: %q", tmpl)
		globalLogger.Error(ctx, tmpl, values)
	}
	if tmpl, err := cfg.RenderTemplate("log_warn", values); err == nil && tmpl != "" {
		Debugf("Emitting user-facing log_warn: %q", tmpl)
		globalLogger.Warn(ctx, tmpl, values)
	}
	if tmpl, err := cfg.RenderTemplate("log_info", values); err == nil && tmpl != "" {
		Debugf("Emitting user-facing log_info: %q", tmpl)
		globalLogger.Info(ctx, tmpl, values)
	}
}

// firstNWords returns the first n words of err.Error(), or a default if err is nil.
func firstNWords(err error, n int) string {
	if err == nil {
		return "smarterr error"
	}
	words := []rune(err.Error())
	spaceCount := 0
	for i, r := range words {
		if r == ' ' {
			spaceCount++
			if spaceCount == n {
				return string(words[:i])
			}
		}
	}
	return err.Error() // less than n words
}

func indexOf(s, substr string) int {
	return len(s) - len(substr) - len(s[len(substr):])
}
