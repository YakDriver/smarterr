package smarterr

import (
	"context"
	"runtime"

	"github.com/YakDriver/smarterr/internal"
	fwdiag "github.com/hashicorp/terraform-plugin-framework/diag"
	sdkdiag "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

var (
	wrappedFS      FileSystem
	wrappedBaseDir string
)

// SetFS allows the host application to provide a FileSystem implementation and the base directory for path normalization.
func SetFS(fs FileSystem, baseDir string) {
	wrappedFS = fs
	wrappedBaseDir = baseDir
}

func AppendFW(ctx context.Context, diags fwdiag.Diagnostics, err error, keyvals ...any) {
	defer func() {
		if r := recover(); r != nil {
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
		diags.AddError(summary, detail)
	}, err, keyvals...)
}

func AppendSDK(ctx context.Context, diags sdkdiag.Diagnostics, err error, keyvals ...any) sdkdiag.Diagnostics {
	defer func() {
		if r := recover(); r != nil {
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
	if wrappedFS == nil {
		addFallbackInitError(add, err)
		return
	}

	relStackPaths := collectRelStackPaths(wrappedBaseDir)
	cfg, cfgErr := internal.LoadConfig(wrappedFS, relStackPaths, wrappedBaseDir)
	if cfgErr != nil {
		addFallbackConfigError(add, err, cfgErr)
		return
	}

	rt := internal.NewRuntime(cfg, err, nil, keyvals...)
	values := rt.BuildTokenValueMap(ctx)

	summary, detail := renderDiagnostics(cfg, err, values)
	add(summary, detail)
	emitLogTemplates(ctx, cfg, values)
}

// addFallbackInitError handles the fallback for missing FS.
func addFallbackInitError(add func(summary, detail string), err error) {
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
	const stackDepth = 5
	pcs := make([]uintptr, stackDepth)
	n := runtime.Callers(2, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	var relStackPaths []string
	for i := 0; i < n; i++ {
		frame, more := frames.Next()
		if frame.File != "" && baseDir != "" {
			idx := indexOf(frame.File, baseDir+"/")
			if idx != -1 {
				rel := frame.File[idx+len(baseDir)+1:]
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
func renderDiagnostics(cfg *internal.Config, err error, values map[string]any) (string, string) {
	summaryTmpl, summaryErr := cfg.RenderTemplate("error_summary", values)
	var summary string
	if summaryErr != nil {
		summary = firstNWords(err, 3)
	} else {
		summary = summaryTmpl
	}
	detailTmpl, detailErr := cfg.RenderTemplate("error_detail", values)
	var detail string
	if detailErr != nil || summaryErr != nil {
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
		return summary, detail
	}
	detail = detailTmpl
	return summary, detail
}

// emitLogTemplates checks for log_error, log_warn, and log_info templates and emits logs if present.
func emitLogTemplates(ctx context.Context, cfg *internal.Config, values map[string]any) {
	if globalLogger == nil {
		return
	}
	if tmpl, err := cfg.RenderTemplate("log_error", values); err == nil && tmpl != "" {
		globalLogger.Error(ctx, tmpl, values)
	}
	if tmpl, err := cfg.RenderTemplate("log_warn", values); err == nil && tmpl != "" {
		globalLogger.Warn(ctx, tmpl, values)
	}
	if tmpl, err := cfg.RenderTemplate("log_info", values); err == nil && tmpl != "" {
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
