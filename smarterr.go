package smarterr

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"

	"github.com/YakDriver/smarterr/internal"
	fwdiag "github.com/hashicorp/terraform-plugin-framework/diag"
	sdkdiag "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

const (
	ID           = "id"
	ResourceName = "resource_name"
	ServiceName  = "service_name"

	DiagnosticSummaryKey = "diagnostic_summary"
	DiagnosticDetailKey  = "diagnostic_detail"
	ErrorSummaryKey      = "error_summary"
	ErrorDetailKey       = "error_detail"
	LogErrorKey          = "log_error"
	LogWarnKey           = "log_warn"
	LogInfoKey           = "log_info"
)

const (
	smarterrContextKey = internal.SmarterrContextKey
)

// Re-export internal.Debugf for internal debugging
var Debugf = internal.Debugf

var (
	wrappedFS      FileSystem
	wrappedBaseDir string
)

var glblCallID atomic.Uint64 // atomic counter for tracing

// SetFS allows the host application to provide a FileSystem implementation and the base directory for path normalization.
func SetFS(fs FileSystem, baseDir string) {
	Debugf("SetFS called with baseDir=%q", baseDir)
	wrappedFS = fs
	wrappedBaseDir = baseDir
}

// EnrichAppend is a helper function that enriches diagnostics with smarterr information.
// This will not change the severity of either incoming or existing diagnostics, but will change
// the summary and detail of _incoming_ diagnostics only with smarterr information.
// Mutates the diagnostics in place via pointer, matching the framework pattern.
//
// Template usage:
//   - If you want to customize the output for framework-generated diagnostics (e.g., value conversion errors),
//     define `diagnostic_summary` and `diagnostic_detail` templates in your config. These will be used by EnrichAppend.
//   - If these templates are not defined, the original diagnostic summary and detail are used.
//   - Note: All output is a diagnostic; the template name refers to the input type (error vs. diagnostic).
func EnrichAppend(ctx context.Context, existing *fwdiag.Diagnostics, incoming fwdiag.Diagnostics, keyvals ...any) {
	ctx, callID := globalCallID(ctx)

	// Debug will NOT be output until LoadConfigWithDiagnostics + the setting in the config
	// enables it because it without config we don't know if debug is enabled. Subsequent
	// calls after config load will show debug if enabled.
	Debugf("[EnrichAppend %s] called with len(incoming): %d, keyvals: %v", callID, len(incoming), keyvals)
	defer func() {
		if r := recover(); r != nil {
			Debugf("[EnrichAppend %s] Panic recovered: %v", callID, r)
			// Fallback: append all incoming diags not already in existing
			for _, diag := range incoming {
				if diag == nil || existing.Contains(diag) {
					continue
				}
				existing.Append(diag)
			}
		}
	}()
	if len(incoming) == 0 {
		return
	}
	// Load config once
	if wrappedFS == nil {
		Debugf("[EnrichAppend %s] No wrappedFS set; cannot enrich diagnostics", callID)
		for _, diag := range incoming {
			if diag == nil || existing.Contains(diag) {
				continue
			}
			existing.Append(diag)
		}
		return
	}
	relStackPaths := collectRelStackPaths(ctx, wrappedBaseDir)
	var diagnostics []error
	cfg, cfgErr := internal.LoadConfigWithDiagnostics(ctx, wrappedFS, relStackPaths, wrappedBaseDir, &diagnostics)
	if cfgErr != nil {
		Debugf("[EnrichAppend %s] Config load error: %v", callID, cfgErr)
		for _, diag := range incoming {
			if diag == nil || existing.Contains(diag) {
				continue
			}
			existing.Append(diag)
		}
		return
	}

	Debugf("[EnrichAppend %s] diagnostics, len(incoming): %d", callID, len(incoming))
	for _, diag := range incoming {
		if diag == nil {
			continue
		}
		// Deduplicate before enrichment
		if existing.Contains(diag) {
			continue
		}
		Debugf("[EnrichAppend %s] enriching diagnostic: %+v", callID, diag)
		// Enrich: build runtime with diagnostic as a token arg
		diagMap := map[string]string{
			"summary":  diag.Summary(),
			"detail":   diag.Detail(),
			"severity": diag.Severity().String(),
		}
		rt := internal.NewRuntimeWithDiagnostics(ctx, cfg, nil, nil, &diagnostics, append(keyvals, "diagnostic", diagMap)...)
		values := rt.BuildTokenValueMap(ctx)
		// Render summary/detail using diagnostic templates if present, else fallback to original
		summary, detail := diag.Summary(), diag.Detail()
		if s, err := cfg.RenderTemplate(ctx, DiagnosticSummaryKey, values); err == nil && s != "" {
			summary = s
		}
		if d, err := cfg.RenderTemplate(ctx, DiagnosticDetailKey, values); err == nil && d != "" {
			detail = d
		}
		// Create enriched diagnostic
		enriched := fwdiag.NewErrorDiagnostic(summary, detail)
		// Deduplicate after enrichment
		if existing.Contains(enriched) {
			continue
		}
		existing.Append(enriched)
	}
}

// AddError adds a formatted error to Terraform Plugin Framework diagnostics.
// Mutates the diagnostics in place via pointer, matching the framework pattern.
//
// Template usage:
//   - To customize the output for errors (Go error values), define `error_summary` and `error_detail` templates in your config.
//   - These templates control the summary and detail for diagnostics created from errors via AddError.
//   - If these templates are not defined, a fallback using the original error is used.
//   - Note: All output is a diagnostic; the template name refers to the input type (error vs. diagnostic).
func AddError(ctx context.Context, diags *fwdiag.Diagnostics, err error, keyvals ...any) {
	ctx, callID := globalCallID(ctx)
	Debugf("[AddError %s] called with error: %v", callID, err)
	defer func() {
		if r := recover(); r != nil {
			Debugf("[AddError %s] Panic recovered: %v", callID, r)
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
		Debugf("[AddError %s] add error: summary=%q detail=%q", callID, summary, detail)
		diags.AddError(summary, detail)
	}, err, keyvals...)
}

// Append adds a formatted error to Terraform Plugin SDK diagnostics and returns the updated diagnostics slice.
//
// Template usage:
//   - To customize the output for errors (Go error values), define `error_summary` and `error_detail` templates in your config.
//   - These templates control the summary and detail for diagnostics created from errors via Append.
//   - If these templates are not defined, a fallback using the original error is used.
//   - Note: All output is a diagnostic; the template name refers to the input type (error vs. diagnostic).
func Append(ctx context.Context, diags sdkdiag.Diagnostics, err error, keyvals ...any) sdkdiag.Diagnostics {
	ctx, callID := globalCallID(ctx)
	Debugf("[Append %s] called with error: %v", callID, err)
	defer func() {
		if r := recover(); r != nil {
			Debugf("[Append %s] Panic recovered: %v", callID, r)
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
		Debugf("[Append %s] add error: summary=%q detail=%q", callID, summary, detail)
		diags = append(diags, sdkdiag.Diagnostic{
			Severity: sdkdiag.Error,
			Summary:  summary,
			Detail:   detail,
		})
	}, err, keyvals...)
	return diags
}

func globalCallID(ctx context.Context) (context.Context, string) {
	callID := ctx.Value(smarterrContextKey)
	callIDStr := ""
	if callID != nil {
		callIDStr, _ = callID.(string)
	} else {
		callIDStr = fmt.Sprintf("%d", glblCallID.Add(1))
		ctx = context.WithValue(ctx, smarterrContextKey, callIDStr)
	}
	return ctx, callIDStr
}

// appendCommon is a shared helper for AddError and Append that resolves and formats error messages
// using the smarterr configuration. It attempts to load configuration from the embedded filesystem and
// the caller's directory, then builds a runtime to render the final error message. If any step fails,
// it appends a fallback error message that always includes the original error (if present) in the summary.
// The add function is used to append the error to the diagnostics in a way appropriate for the caller.
func appendCommon(ctx context.Context, add func(summary, detail string), err error, keyvals ...any) {
	ctx, callID := globalCallID(ctx)
	Debugf("[appendCommon %s] called with error: %v, keyvals: %v", callID, err, keyvals)
	var diagnostics []error
	if wrappedFS == nil {
		Debugf("[appendCommon %s] No wrappedFS set; calling addFallbackInitError", callID)
		addFallbackInitError(add, err)
		return
	}

	relStackPaths := collectRelStackPaths(ctx, wrappedBaseDir)
	Debugf("[appendCommon %s] collectRelStackPaths returned: %v", callID, relStackPaths)
	cfg, cfgErr := internal.LoadConfigWithDiagnostics(ctx, wrappedFS, relStackPaths, wrappedBaseDir, &diagnostics)
	if cfgErr != nil {
		Debugf("[appendCommon %s] Config load error: %v", callID, cfgErr)
		addFallbackConfigError(add, err, cfgErr)
		return
	}

	rt := internal.NewRuntimeWithDiagnostics(ctx, cfg, err, nil, &diagnostics, keyvals...)
	values := rt.BuildTokenValueMap(ctx)

	summary, detail := renderDiagnosticsWithDiagnostics(ctx, cfg, err, values, &diagnostics)
	Debugf("[appendCommon %s] renderDiagnostics returned summary=%q detail=%q", callID, summary, detail)
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
func collectRelStackPaths(ctx context.Context, baseDir string) []string {
	ctx, callID := globalCallID(ctx)
	Debugf("[collectRelStackPaths %s] called with baseDir=%q", callID, baseDir)
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
func renderDiagnosticsWithDiagnostics(ctx context.Context, cfg *internal.Config, err error, values map[string]any, diagnostics *[]error) (string, string) {
	ctx, callID := globalCallID(ctx)
	Debugf("[renderDiagnostics %s] called with error: %v, values: %v", callID, err, values)
	summaryTmpl, summaryErr := cfg.RenderTemplate(ctx, ErrorSummaryKey, values)
	var summary string
	if summaryErr != nil {
		Debugf("Summary template error: %v", summaryErr)
		*diagnostics = append(*diagnostics, summaryErr)
		summary = firstNWords(err, 3)
	} else {
		summary = summaryTmpl
	}
	detailTmpl, detailErr := cfg.RenderTemplate(ctx, ErrorDetailKey, values)
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
	ctx, callID := globalCallID(ctx)
	Debugf("[emitLogTemplates %s] called", callID)
	if globalLogger == nil {
		Debugf("[emitLogTemplates %s] No globalLogger set; skipping user-facing log emission")
		return
	}
	if tmpl, err := cfg.RenderTemplate(ctx, LogErrorKey, values); err == nil && tmpl != "" {
		Debugf("[emitLogTemplates %s] Emitting user-facing log_error: %q", tmpl)
		globalLogger.Error(ctx, tmpl, values)
	}
	if tmpl, err := cfg.RenderTemplate(ctx, LogWarnKey, values); err == nil && tmpl != "" {
		Debugf("[emitLogTemplates %s] Emitting user-facing log_warn: %q", tmpl)
		globalLogger.Warn(ctx, tmpl, values)
	}
	if tmpl, err := cfg.RenderTemplate(ctx, LogInfoKey, values); err == nil && tmpl != "" {
		Debugf("[emitLogTemplates %s] Emitting user-facing log_info: %q", tmpl)
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
