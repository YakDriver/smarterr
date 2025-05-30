// internal/runtime.go

package internal

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"text/template"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type Runtime struct {
	Config *Config
	Args   map[string]any
	Error  error
	Diags  diag.Diagnostics
	Logger Logger
}

func NewRuntime(cfg *Config, err error, logger Logger, kv ...any) *Runtime {
	// Use the provided logger or fall back to the global logger
	if logger == nil {
		logger = GetGlobalLogger()
	}

	// Parse key-value pairs
	args := parseKeyvals(kv...)

	// Log if the configuration or error is nil
	if cfg == nil {
		logger.Error("Runtime configuration is nil")
	}

	if err != nil {
		logger.Error("Runtime initialized with error: %v", err)
	}

	return &Runtime{
		Config: cfg,
		Error:  err,
		Args:   args,
		Logger: logger,
	}
}

// applyTransforms applies named transforms (from config) to a value, in order.
func (rt *Runtime) applyTransforms(token *Token, value string) string {
	if len(token.Transforms) == 0 || rt.Config == nil {
		return value
	}
	for _, tname := range token.Transforms {
		var tdef *Transform
		for i := range rt.Config.Transforms {
			if rt.Config.Transforms[i].Name == tname {
				tdef = &rt.Config.Transforms[i]
				break
			}
		}
		if tdef == nil {
			continue // skip missing transforms
		}
		for _, step := range tdef.Steps {
			switch step.Type {
			case "strip_prefix":
				value = applyStripPrefix(value, step)
			case "strip_suffix":
				value = applyStripSuffix(value, step)
			case "remove":
				value = applyRemove(value, step)
			case "replace":
				value = applyReplace(value, step)
			case "trim_space":
				value = strings.TrimSpace(value)
			case "fix_space":
				value = strings.TrimSpace(value)
				value = regexp.MustCompile(`\s+`).ReplaceAllString(value, " ")
			case "lower":
				value = strings.ToLower(value)
			case "upper":
				value = strings.ToUpper(value)
				// Add more transform types as needed
			}
		}
	}
	return value
}

// Helper for strip_prefix
func applyStripPrefix(value string, step TransformStep) string {
	value = strings.TrimSpace(value)
	if step.Value == nil {
		return value
	}
	if step.Recurse != nil && *step.Recurse {
		for {
			if strings.HasPrefix(value, *step.Value) {
				value = strings.TrimPrefix(value, *step.Value)
				value = strings.TrimSpace(value)
				continue
			}
			break
		}
		return value
	}
	if strings.HasPrefix(value, *step.Value) {
		value = strings.TrimPrefix(value, *step.Value)
		value = strings.TrimSpace(value)
	}
	return value
}

// Helper for strip_suffix
func applyStripSuffix(value string, step TransformStep) string {
	value = strings.TrimSpace(value)
	if step.Value == nil {
		return value
	}
	if step.Recurse != nil && *step.Recurse {
		for {
			if strings.HasSuffix(value, *step.Value) {
				value = strings.TrimSuffix(value, *step.Value)
				value = strings.TrimSpace(value)
				continue
			}
			break
		}
		return value
	}
	if strings.HasSuffix(value, *step.Value) {
		value = strings.TrimSuffix(value, *step.Value)
		value = strings.TrimSpace(value)
	}
	return value
}

// Helper for remove
func applyRemove(value string, step TransformStep) string {
	if step.Regex != nil {
		if step.Recurse != nil && *step.Recurse {
			for {
				re := regexp.MustCompile(*step.Regex)
				newValue := re.ReplaceAllString(value, "")
				if newValue == value {
					break
				}
				value = newValue
			}
			return value
		}
		re := regexp.MustCompile(*step.Regex)
		return re.ReplaceAllString(value, "")
	}
	if step.Value != nil {
		if step.Recurse != nil && *step.Recurse {
			for {
				newValue := strings.ReplaceAll(value, *step.Value, "")
				if newValue == value {
					break
				}
				value = newValue
			}
			return value
		}
		return strings.ReplaceAll(value, *step.Value, "")
	}
	return value
}

// Helper for replace
func applyReplace(value string, step TransformStep) string {
	if step.Regex != nil && step.With != nil {
		if step.Recurse != nil && *step.Recurse {
			for {
				re := regexp.MustCompile(*step.Regex)
				newValue := re.ReplaceAllString(value, *step.With)
				if newValue == value {
					break
				}
				value = newValue
			}
			return value
		}
		re := regexp.MustCompile(*step.Regex)
		return re.ReplaceAllString(value, *step.With)
	}
	return value
}

// Resolve takes a token and resolves it based on the runtime information.
// It supports various source types such as parameters, context values,
// error inspection, call stack inspection, and runtime arguments.
func (t *Token) Resolve(ctx context.Context, rt *Runtime) string {
	logger := rt.Logger

	// Infer source if not set
	source := t.Source
	if source == "" {
		switch {
		case t.Parameter != nil:
			source = "parameter"
		case t.Context != nil:
			source = "context"
		case t.Arg != nil:
			source = "arg"
		case len(t.StackMatches) > 0:
			source = "call_stack"
		default:
			source = "parameter" // fallback for backward compatibility
		}
	}

	var value string

	switch source {
	case "parameter":
		// Look up the parameter by name.
		if t.Parameter == nil {
			logger.Error("Token %q: parameter name must be set when source is 'parameter'", t.Name)
			value = fallbackMessage(rt.Config, t.Name)
		} else {
			for _, p := range rt.Config.Parameters {
				if p.Name == *t.Parameter {
					value = p.Value
					break
				}
			}
			if value == "" {
				logger.Warn("Token %q: parameter %q not found", t.Name, *t.Parameter)
				value = fallbackMessage(rt.Config, t.Name)
			}
		}

	case "context":
		// Extract value from context by key.
		if t.Context == nil {
			logger.Error("Token %q: context key must be set when source is 'context'", t.Name)
			value = fallbackMessage(rt.Config, t.Name)
		} else {
			val := ctx.Value(*t.Context)
			if val == nil {
				logger.Warn("Token %q: context value for key %q not found", t.Name, *t.Context)
				value = fallbackMessage(rt.Config, t.Name)
			} else {
				value = fmt.Sprintf("%v", val)
			}
		}

	case "call_stack":
		// Filter StackMatches based on Token.StackMatches
		var filteredStackMatches []StackMatch
		for _, name := range t.StackMatches {
			for _, sm := range rt.Config.StackMatches {
				if sm.Name == name {
					filteredStackMatches = append(filteredStackMatches, sm)
					break
				}
			}
		}

		// Gather the call stack
		frames, err := gatherCallStack(3) // Skip 3 frames to exclude runtime.Callers, gatherCallStack, and Resolve
		if err != nil {
			logger.Error("Token %q: error gathering call stack: %v", t.Name, err)
			value = fallbackMessage(rt.Config, t.Name)
		} else {
			// Process the filtered stack matches
			display, err := processStackMatches(filteredStackMatches, frames)
			if err != nil {
				logger.Error("Token %q: error processing stack matches: %v", t.Name, err)
				value = fallbackMessage(rt.Config, t.Name)
			} else if display != "" {
				value = display
			} else {
				logger.Warn("Token %q: no match found in call stack", t.Name)
				value = ""
			}
		}

	case "error":
		// Inspect the error (possibly via reflection or a known error interface).
		if rt.Error == nil {
			logger.Warn("Token %q: no error provided in runtime", t.Name)
			value = fallbackMessage(rt.Config, t.Name)
		} else {
			value = fmt.Sprintf("%s", rt.Error)
		}

	case "arg":
		// Pull from runtime arguments.
		if t.Arg == nil {
			logger.Error("Token %q: arg key must be set when source is 'arg'", t.Name)
			value = fallbackMessage(rt.Config, t.Name)
		} else {
			argVal, ok := rt.Args[*t.Arg]
			if !ok {
				logger.Warn("Token %q: argument not found in runtime args", t.Name)
				value = fallbackMessage(rt.Config, t.Name)
			} else {
				value = fmt.Sprintf("%v", argVal)
			}
		}

	default:
		logger.Error("Token %q: unknown source %q", t.Name, source)
		value = fallbackMessage(rt.Config, t.Name)
	}

	// Only apply transforms if t.Transforms is non-nil and non-empty
	if t.Transforms != nil && len(t.Transforms) > 0 {
		value = rt.applyTransforms(t, value)
	}
	return value
}

// BuildTokenValueMap resolves all tokens in the config and returns a map of token name to value.
func (rt *Runtime) BuildTokenValueMap(ctx context.Context) map[string]any {
	values := make(map[string]any)
	for _, t := range rt.Config.Tokens {
		values[t.Name] = t.Resolve(ctx, rt)
	}
	return values
}

// gatherCallStack retrieves the call stack frames, skipping the specified number of frames.
func gatherCallStack(skip int) ([]runtime.Frame, error) {
	callers := make([]uintptr, 10) // Adjust size as needed
	n := runtime.Callers(skip, callers)
	if n == 0 {
		return nil, fmt.Errorf("no call stack available")
	}

	frames := runtime.CallersFrames(callers[:n])
	var result []runtime.Frame
	for {
		frame, more := frames.Next()
		result = append(result, frame)
		if !more {
			break
		}
	}
	return result, nil
}

// processStackMatches processes the stack frames and matches them against the StackMatch rules.
// If a match is found, it returns the Display value of the matching rule.
// Note, if neither CalledFrom nor CalledAfter is specified, it will match and return display.
func processStackMatches(stackMatches []StackMatch, frames []runtime.Frame) (string, error) {
	var previousFunc string

	// Iterate through the stack frames
	for _, frame := range frames {
		// Check each StackMatch rule
		for _, match := range stackMatches {
			// Match CalledFrom if specified
			if match.CalledFrom != "" {
				matched, err := regexp.MatchString(match.CalledFrom, frame.Function)
				if err != nil {
					return "", fmt.Errorf("invalid regex in CalledFrom for StackMatch %q: %w", match.Name, err)
				}
				if !matched {
					continue
				}
			}

			// Match CalledAfter if specified
			if match.CalledAfter != "" {
				matched, err := regexp.MatchString(match.CalledAfter, previousFunc)
				if err != nil {
					return "", fmt.Errorf("invalid regex in CalledAfter for StackMatch %q: %w", match.Name, err)
				}
				if !matched {
					continue
				}
			}

			// If both conditions match (or are not specified), return the Display value
			return match.Display, nil
		}

		// Update the previous function name for the next iteration
		previousFunc = frame.Function
	}

	// No match found
	return "", nil
}

// parseKeyvals parses the provided key-value pairs into a map[string]any.
// This lays the foundation for flexible calling without requiring devs to manually build a map.
// For example, if kv is "id", "rds", "service", "Provider", it will return
// a map with the following structure:
//
//	{
//		"id": "rds",
//		"service": "Provider",
//	}
//
// It ensures that the kv length is even and that all keys are strings.
// If the length is not even or a key is not a string, it panics.
func parseKeyvals(kv ...any) map[string]any {
	logger := GetGlobalLogger()

	// Check if the length of kv is odd
	if len(kv)%2 != 0 {
		logger.Warn("Odd number of key-value arguments: dropping the last key-value pair")
		kv = kv[:len(kv)-1] // Remove the last element
	}

	result := make(map[string]any)

	for i := 0; i < len(kv); i += 2 {
		// Assert that the key is a string
		key, ok := kv[i].(string)
		if !ok {
			logger.Error("Invalid key type at index %d: expected string, got %T", i, kv[i])
			return map[string]any{}
		}

		// Add the key-value pair to the map
		result[key] = kv[i+1]
	}

	return result
}

// RenderTemplate renders a named template from the config using the provided token values.
func (cfg *Config) RenderTemplate(name string, values map[string]any) (string, error) {
	var tmplStr string
	for _, tmpl := range cfg.Templates {
		if tmpl.Name == name {
			tmplStr = tmpl.Format
			break
		}
	}
	if tmplStr == "" {
		return "", fmt.Errorf("template %q not found", name)
	}
	tmpl, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, values)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
