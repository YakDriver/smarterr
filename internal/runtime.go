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
	"text/template/parse"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type Runtime struct {
	Config *Config
	Args   map[string]any
	Error  error
	Diags  diag.Diagnostics
}

func NewRuntime(cfg *Config, err error, _ interface{}, kv ...any) *Runtime {
	// Parse key-value pairs
	args := parseKeyvals(kv...)
	// Emit debug output if config or error is nil
	if cfg == nil {
		Debugf("Runtime configuration is nil")
	}
	if err != nil {
		Debugf("Runtime initialized with error: %v", err)
	}
	return &Runtime{
		Config: cfg,
		Error:  err,
		Args:   args,
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
			value = fallbackMessage(rt.Config, t.Name)
		} else {
			for _, p := range rt.Config.Parameters {
				if p.Name == *t.Parameter {
					value = p.Value
					break
				}
			}
			if value == "" {
				value = fallbackMessage(rt.Config, t.Name)
			}
		}

	case "context":
		// Extract value from context by key.
		if t.Context == nil {
			value = fallbackMessage(rt.Config, t.Name)
		} else {
			val := ctx.Value(*t.Context)
			if val == nil {
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
			value = fallbackMessage(rt.Config, t.Name)
		} else {
			// Process the filtered stack matches
			display, err := processStackMatches(filteredStackMatches, frames)
			if err != nil {
				value = fallbackMessage(rt.Config, t.Name)
			} else if display != "" {
				value = display
			} else {
				value = ""
			}
		}

	case "error":
		// If the token name is "message" or empty, return the error string; otherwise, fallback.
		if rt.Error == nil {
			value = fallbackMessage(rt.Config, t.Name)
		} else if t.Name == "message" || t.Name == "error" || t.Name == "" {
			value = fmt.Sprintf("%s", rt.Error)
		} else {
			value = fallbackMessage(rt.Config, t.Name)
		}

	case "arg":
		// Pull from runtime arguments.
		if t.Arg == nil {
			value = fallbackMessage(rt.Config, t.Name)
		} else {
			argVal, ok := rt.Args[*t.Arg]
			if !ok {
				value = fallbackMessage(rt.Config, t.Name)
			} else {
				value = fmt.Sprintf("%v", argVal)
			}
		}

	default:
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
	// Check if the length of kv is odd
	if len(kv)%2 != 0 {
		Debugf("Odd number of key-value arguments: dropping the last key-value pair")
		kv = kv[:len(kv)-1] // Remove the last element
	}
	result := make(map[string]any)
	for i := 0; i < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			Debugf("Invalid key type at index %d: expected string, got %T", i, kv[i])
			return map[string]any{}
		}
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

	// Scan the template AST for all referenced variables
	vars := collectTemplateVariables(tmpl)
	// Pre-populate missing values with fallback
	for _, v := range vars {
		if _, ok := values[v]; !ok {
			values[v] = fallbackMessage(cfg, v)
		}
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, values)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// collectTemplateVariables walks the template AST and returns a list of all variable names referenced.
func collectTemplateVariables(tmpl *template.Template) []string {
	vars := make(map[string]struct{})
	for _, t := range tmpl.Templates() {
		walkNodes(t.Tree.Root, vars)
	}
	result := make([]string, 0, len(vars))
	for v := range vars {
		result = append(result, v)
	}
	return result
}

// walkNodes recursively walks template nodes and collects variable names.
func walkNodes(node parse.Node, vars map[string]struct{}) {
	switch n := node.(type) {
	case *parse.ListNode:
		for _, child := range n.Nodes {
			walkNodes(child, vars)
		}
	case *parse.ActionNode:
		walkNodes(n.Pipe, vars)
	case *parse.PipeNode:
		for _, cmd := range n.Cmds {
			walkNodes(cmd, vars)
		}
	case *parse.CommandNode:
		for _, arg := range n.Args {
			walkNodes(arg, vars)
		}
	case *parse.FieldNode:
		if len(n.Ident) > 0 {
			vars[n.Ident[0]] = struct{}{}
		}
	case *parse.VariableNode:
		if len(n.Ident) > 0 {
			vars[n.Ident[0]] = struct{}{}
		}
	case *parse.IfNode:
		walkNodes(n.Pipe, vars)
		walkNodes(n.List, vars)
		if n.ElseList != nil {
			walkNodes(n.ElseList, vars)
		}
	case *parse.RangeNode:
		walkNodes(n.Pipe, vars)
		walkNodes(n.List, vars)
		if n.ElseList != nil {
			walkNodes(n.ElseList, vars)
		}
	case *parse.WithNode:
		walkNodes(n.Pipe, vars)
		walkNodes(n.List, vars)
		if n.ElseList != nil {
			walkNodes(n.ElseList, vars)
		}
		// Add more node types as needed
	}
}

func fallbackMessage(cfg *Config, tokenName string) string {
	mode := "empty"
	if cfg != nil && cfg.TokenErrorMode != "" {
		mode = cfg.TokenErrorMode
	}
	switch mode {
	case "detailed":
		return fmt.Sprintf("[unresolved token: %s]", tokenName)
	case "placeholder":
		return fmt.Sprintf("<%s>", tokenName)
	case "empty":
		fallthrough
	default:
		return ""
	}
}
