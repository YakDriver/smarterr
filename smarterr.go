package smarterr

import (
	"context"
	"runtime"
	"strings"

	"github.com/YakDriver/smarterr/filesystem"
	"github.com/YakDriver/smarterr/internal"
	fwdiag "github.com/hashicorp/terraform-plugin-framework/diag"
	sdkdiag "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

var (
	wrappedFS      filesystem.FileSystem
	wrappedBaseDir string
)

// SetFS allows the host application to provide a FileSystem implementation and the base directory for path normalization.
func SetFS(fs filesystem.FileSystem, baseDir string) {
	wrappedFS = fs
	wrappedBaseDir = baseDir
}

func AppendFW(ctx context.Context, diags fwdiag.Diagnostics, err error, keyvals ...any) {
	defer func() {
		if r := recover(); r != nil {
			// Log the panic, append a fallback error message
			msg := "smarterr panic: "
			switch v := r.(type) {
			case error:
				msg += v.Error()
			case string:
				msg += v
			default:
				msg += "unknown panic"
			}
			// Add the original error if present
			if err != nil {
				msg += "; original error: " + err.Error()
			}
			diags.AddError("smarterr Internal Panic", msg)
		}
	}()
	appendCommon(ctx, func(summary, detail string) {
		diags.AddError(summary, detail)
	}, err, keyvals...)
}

func AppendSDK(ctx context.Context, diags sdkdiag.Diagnostics, err error, keyvals ...any) sdkdiag.Diagnostics {
	defer func() {
		if r := recover(); r != nil {
			msg := "smarterr panic: "
			switch v := r.(type) {
			case error:
				msg += v.Error()
			case string:
				msg += v
			default:
				msg += "unknown panic"
			}
			if err != nil {
				msg += "; original error: " + err.Error()
			}
			diags = append(diags, sdkdiag.Diagnostic{
				Severity: sdkdiag.Error,
				Summary:  "smarterr Internal Panic",
				Detail:   msg,
			})
		}
	}()
	appendCommon(ctx, func(summary, detail string) {
		diags = append(diags, sdkdiag.Diagnostic{
			Severity: sdkdiag.Error,
			Summary:  summary,
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
		summary := ""
		if err != nil {
			summary = err.Error() + "; "
		}
		summary += "smarterr initialization: Embedded filesystem not set, use SetFS()"
		add(summary, "")
		return
	}

	// Collect and normalize all call stack file paths relative to wrappedBaseDir
	const stackDepth = 5
	pcs := make([]uintptr, stackDepth)
	n := runtime.Callers(2, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	var relStackPaths []string
	for i := 0; i < n; i++ {
		frame, more := frames.Next()
		if frame.File != "" && wrappedBaseDir != "" {
			idx := indexOf(frame.File, wrappedBaseDir+"/")
			if idx != -1 {
				rel := frame.File[idx+len(wrappedBaseDir)+1:]
				relStackPaths = append(relStackPaths, rel)
			}
		}
		if !more {
			break
		}
	}

	// Use the new reverse-matching config loader with all stack paths
	cfg, cfgErr := internal.LoadConfig(wrappedFS, relStackPaths, wrappedBaseDir)
	if cfgErr != nil {
		summary := ""
		if err != nil {
			summary = err.Error() + "; "
		}
		summary += "smarterr Configuration Error: " + cfgErr.Error()
		add(summary, "")
		return
	}

	rt := internal.NewRuntime(cfg, err, nil, keyvals...)
	values := rt.BuildTokenValueMap(ctx)

	// Use the error_summary template if present, else fallback to old logic
	var rendered string
	if tmpl, err := cfg.RenderTemplate("error_summary", values); err == nil {
		rendered = tmpl
	} else {
		// fallback: join all token values for backward compatibility
		var summary string
		for _, v := range values {
			if s, ok := v.(string); ok && s != "" {
				summary += s + " "
			}
		}
		rendered = strings.TrimSpace(summary)
	}
	add(rendered, "")
}

func indexOf(s, substr string) int {
	return len(s) - len(substr) - len(s[len(substr):])
}
