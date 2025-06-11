package smarterr

import (
	"context"
	"fmt"
	"log"

	tflog "github.com/hashicorp/terraform-plugin-log/tflog"
)

// Logger is a generic logging interface for smarterr.
type Logger interface {
	Debug(ctx context.Context, msg string, keyvals map[string]any)
	Info(ctx context.Context, msg string, keyvals map[string]any)
	Warn(ctx context.Context, msg string, keyvals map[string]any)
	Error(ctx context.Context, msg string, keyvals map[string]any)
}

var globalLogger Logger

// SetLogger sets the global logger for smarterr.
func SetLogger(logger Logger) {
	globalLogger = logger
}

// StdLogger is an adapter for the standard Go log package.
type StdLogger struct{}

func (l StdLogger) Debug(ctx context.Context, msg string, keyvals map[string]any) {
	logPrint("DEBUG", msg, keyvals)
}
func (l StdLogger) Info(ctx context.Context, msg string, keyvals map[string]any) {
	logPrint("INFO", msg, keyvals)
}
func (l StdLogger) Warn(ctx context.Context, msg string, keyvals map[string]any) {
	logPrint("WARN", msg, keyvals)
}
func (l StdLogger) Error(ctx context.Context, msg string, keyvals map[string]any) {
	logPrint("ERROR", msg, keyvals)
}

func logPrint(level, msg string, keyvals map[string]any) {
	logMsg := level + ": " + msg
	if len(keyvals) > 0 {
		logMsg += " | "
		for k, v := range keyvals {
			logMsg += k + "=" + fmt.Sprint(v) + " "
		}
	}
	log.Println(logMsg)
}

// TFLogLogger is an adapter for tflog (Terraform plugin log).
type TFLogLogger struct{}

func (l TFLogLogger) Debug(ctx context.Context, msg string, keyvals map[string]any) {
	fmt.Printf("TFLog Debug: %s\n", msg)
	tflog.Debug(ctx, msg, keyvals)
}
func (l TFLogLogger) Info(ctx context.Context, msg string, keyvals map[string]any) {
	fmt.Printf("TFLog Info: %s\n", msg)
	tflog.Info(ctx, msg, keyvals)
}
func (l TFLogLogger) Warn(ctx context.Context, msg string, keyvals map[string]any) {
	fmt.Printf("TFLog Warn: %s\n", msg)
	tflog.Warn(ctx, msg, keyvals)
}
func (l TFLogLogger) Error(ctx context.Context, msg string, keyvals map[string]any) {
	fmt.Printf("TFLog Error: %s\n", msg)
	tflog.Error(ctx, msg, keyvals)
}
