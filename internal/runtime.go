// internal/runtime.go

package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type Runtime struct {
	Config     *Config
	Args       map[string]any
	Error      error
	Diagnostic diag.Diagnostic // single diagnostic for enrichment context
}

func NewRuntime(ctx context.Context, cfg *Config, err error, kv ...any) *Runtime {
	callID := globalCallID(ctx)

	// Parse key-value pairs
	args := parseKeyvals(ctx, kv...)
	// Emit debug output if config or error is nil
	if cfg == nil {
		Debugf("[NewRuntime %s] Runtime configuration is nil", callID)
	}
	if err != nil {
		Debugf("[NewRuntime %s] Runtime initialized with error: %v", callID, err)
	}
	return &Runtime{
		Config: cfg,
		Error:  err,
		Args:   args,
	}
}

func NewRuntimeForDiagnostic(ctx context.Context, cfg *Config, diagnostic diag.Diagnostic, kv ...any) *Runtime {
	callID := globalCallID(ctx)
	args := parseKeyvals(ctx, kv...)
	if cfg == nil {
		Debugf("[NewRuntimeForDiagnostic %s] Runtime configuration is nil", callID)
	}
	if diagnostic != nil {
		Debugf("[NewRuntimeForDiagnostic %s] Runtime initialized with args (%+v) and diagnostic: %v", callID, args, diagnostic)
	}

	return &Runtime{
		Config:     cfg,
		Diagnostic: diagnostic,
		Args:       args,
	}
}

// applyTransforms applies named transforms (from config) to a value, in order.
func (rt *Runtime) applyTransforms(ctx context.Context, token *Token, value string) string {
	callID := globalCallID(ctx)
	if len(token.Transforms) == 0 || rt.Config == nil {
		Debugf("[applyTransforms %s] No transforms (%d) or Config (%t)", callID, len(token.Transforms), rt.Config == nil)
		return value
	}
	Debugf("[applyTransforms %s] to token %q: %v", callID, token.Name, token.Transforms)
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
	Debugf("[applyTransforms %s] %s transformed value: %q", callID, token.Name, value)
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
			if after, ok := strings.CutPrefix(value, *step.Value); ok {
				value = after
				value = strings.TrimSpace(value)
				continue
			}
			break
		}
		return value
	}
	if after, ok := strings.CutPrefix(value, *step.Value); ok {
		value = after
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

func globalCallID(ctx context.Context) string {
	var callID string
	if v := ctx.Value(any("smarterrCallID")); v != nil {
		callID, _ = v.(string)
	}
	return callID
}

// Resolve takes a token and resolves it based on the runtime information.
// It supports various source types such as parameters, context values,
// error inspection, call stack inspection, and runtime arguments.
func (t *Token) Resolve(ctx context.Context, rt *Runtime) any {
	callID := globalCallID(ctx)
	Debugf("[Token.Resolve %s] Resolving token: %s, source: %s, parameter: %v, context: %v, arg: %v, stack_matches: %v",
		callID, t.Name, t.Source, t.Parameter, t.Context, t.Arg, t.StackMatches)
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

	switch source {
	case "diagnostic":
		if rt.Diagnostic != nil {
			diag := rt.Diagnostic
			result := make(map[string]any)
			result["summary"] = diag.Summary()
			result["detail"] = diag.Detail()
			result["severity"] = diag.Severity().String()
			// Apply field transforms if present
			if t.FieldTransforms != nil {
				for field, transforms := range t.FieldTransforms {
					if val, ok := result[field].(string); ok {
						for _, tname := range transforms {
							val = rt.applyTransformByName(tname, val)
						}
						result[field] = val
					}
				}
			}
			return result
		}
		Debugf("[Token.Resolve %s] Fallback for token %q: diagnostic info not found in Runtime.Diagnostic", callID, t.Name)
		return map[string]any{
			"summary":  fallbackMessage(rt.Config, t.Name+".summary", "diagnostic summary not found"),
			"detail":   fallbackMessage(rt.Config, t.Name+".detail", "diagnostic detail not found"),
			"severity": fallbackMessage(rt.Config, t.Name+".severity", "diagnostic severity not found"),
		}
	case "parameter":
		var value string
		if t.Parameter == nil {
			Debugf("[Token.Resolve %s] Fallback for token %q: token.Parameter is nil", callID, t.Name)
			value = fallbackMessage(rt.Config, t.Name, "token.Parameter is nil")
		} else {
			for _, p := range rt.Config.Parameters {
				if p.Name == *t.Parameter {
					value = p.Value
					break
				}
			}
			if value == "" {
				Debugf("[Token.Resolve %s] Fallback for token %q: parameter not found in config", callID, t.Name)
				value = fallbackMessage(rt.Config, t.Name, "parameter not found in config")
			}
		}
		if t.Transforms != nil && len(t.Transforms) > 0 {
			value = rt.applyTransforms(ctx, t, value)
		}
		return value
	case "context":
		var value string
		if t.Context == nil {
			Debugf("[Token.Resolve %s] Fallback for token %q: token.Context is nil", callID, t.Name)
			value = fallbackMessage(rt.Config, t.Name, "token.Context is nil")
		} else {
			val := ctx.Value(*t.Context)
			if val == nil {
				Debugf("[Token.Resolve %s] Fallback for token %q: context value is nil", callID, t.Name)
				value = fallbackMessage(rt.Config, t.Name, "context value is nil")
			} else {
				value = fmt.Sprintf("%v", val)
			}
		}
		if t.Transforms != nil && len(t.Transforms) > 0 {
			value = rt.applyTransforms(ctx, t, value)
		}
		return value
	case "call_stack":
		var value string
		var filteredStackMatches []StackMatch
		for _, name := range t.StackMatches {
			for _, sm := range rt.Config.StackMatches {
				if sm.Name == name {
					filteredStackMatches = append(filteredStackMatches, sm)
					break
				}
			}
		}
		frames, err := gatherCallStack(3)
		if err != nil {
			Debugf("[Token.Resolve %s] Fallback for token %q: call stack unavailable", callID, t.Name)
			value = fallbackMessage(rt.Config, t.Name, "call stack unavailable")
		} else {
			display, err := processStackMatches(filteredStackMatches, frames)
			if err != nil {
				Debugf("[Token.Resolve %s] Fallback for token %q: stack match error: %s", callID, t.Name, err.Error())
				value = fallbackMessage(rt.Config, t.Name, "stack match error: "+err.Error())
			} else if display != "" {
				value = display
			} else {
				Debugf("[Token.Resolve %s] Fallback for token %q: no stack match found", callID, t.Name)
				value = fallbackMessage(rt.Config, t.Name, "no stack match found")
			}
		}
		if t.Transforms != nil && len(t.Transforms) > 0 {
			value = rt.applyTransforms(ctx, t, value)
		}
		return value
	case "error_stack":
		var value string
		var filteredStackMatches []StackMatch
		for _, name := range t.StackMatches {
			for _, sm := range rt.Config.StackMatches {
				if sm.Name == name {
					filteredStackMatches = append(filteredStackMatches, sm)
					break
				}
			}
		}
		var frames []runtime.Frame
		Debugf("[Token.Resolve %s] err type: %T", callID, rt.Error)
		var stackProvider interface{ Stack() []runtime.Frame }
		if errors.As(rt.Error, &stackProvider) && stackProvider != nil {
			frames = stackProvider.Stack()
		}
		if len(frames) == 0 {
			Debugf("[Token.Resolve %s] Fallback for token %q: error_stack unavailable", callID, t.Name)
			value = fallbackMessage(rt.Config, t.Name, "error_stack unavailable")
		} else {
			display, err := processStackMatches(filteredStackMatches, frames)
			if err != nil {
				Debugf("[Token.Resolve %s] Fallback for token %q: error_stack match error: %s", callID, t.Name, err.Error())
				value = fallbackMessage(rt.Config, t.Name, "error_stack match error: "+err.Error())
			} else if display != "" {
				value = display
			} else {
				Debugf("[Token.Resolve %s] Fallback for token %q: no error_stack match found", callID, t.Name)
				value = fallbackMessage(rt.Config, t.Name, "no error_stack match found")
			}
		}
		if t.Transforms != nil && len(t.Transforms) > 0 {
			value = rt.applyTransforms(ctx, t, value)
		}
		return value
	case "error":
		var value string
		Debugf("[Token.Resolve %s] Resolving error token: %s, err: %s", callID, t.Name, rt.Error)
		if rt.Error == nil {
			Debugf("[Token.Resolve %s] Fallback for token %q: rt.Error is nil", callID, t.Name)
			value = fallbackMessage(rt.Config, t.Name, "rt.Error is nil")
		} else {
			value = fmt.Sprintf("%s", rt.Error)
		}
		if t.Transforms != nil && len(t.Transforms) > 0 {
			value = rt.applyTransforms(ctx, t, value)
		}
		return value
	case "arg":
		var value string
		if t.Arg == nil {
			Debugf("[Token.Resolve %s] Fallback for token %q: token.Arg is nil", callID, t.Name)
			value = fallbackMessage(rt.Config, t.Name, "token.Arg is nil")
		} else {
			argVal, ok := rt.Args[*t.Arg]
			if !ok {
				Debugf("[Token.Resolve %s] Fallback for token %q: argument (%s) not found in runtime args", callID, t.Name, *t.Arg)
				value = fallbackMessage(rt.Config, t.Name, fmt.Sprintf("argument (%s) not found in runtime args", *t.Arg))
			} else {
				value = fmt.Sprintf("%v", argVal)
			}
		}
		if t.Transforms != nil && len(t.Transforms) > 0 {
			value = rt.applyTransforms(ctx, t, value)
		}
		return value
	case "hints":
		var value string
		Debugf("[Token.Resolve %s] Resolving hints token: %s", callID, t.Name)
		if rt.Error != nil {
			value = resolveHints(ctx, rt.Error.Error(), rt.Config)
		}
		if value == "" {
			Debugf("[Token.Resolve %s] Fallback for token %q: no matching hint found", callID, t.Name)
			value = fallbackMessage(rt.Config, t.Name, "no matching hint found")
		}
		if t.Transforms != nil && len(t.Transforms) > 0 {
			value = rt.applyTransforms(ctx, t, value)
		}
		return value
	default:
		var value string
		Debugf("[Token.Resolve %s] Fallback for token %q: unknown token source", callID, t.Name)
		value = fallbackMessage(rt.Config, t.Name, "unknown token source")
		if t.Transforms != nil && len(t.Transforms) > 0 {
			value = rt.applyTransforms(ctx, t, value)
		}
		return value
	}
}

// Helper to apply a named transform to a value (for field transforms)
func (rt *Runtime) applyTransformByName(name, value string) string {
	if rt.Config == nil {
		return value
	}
	for i := range rt.Config.Transforms {
		if rt.Config.Transforms[i].Name == name {
			for _, step := range rt.Config.Transforms[i].Steps {
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
				}
			}
			break
		}
	}
	return value
}

// BuildTokenValueMap resolves all tokens in the config and returns a map of token name to value.
func (rt *Runtime) BuildTokenValueMap(ctx context.Context) map[string]any {
	callID := globalCallID(ctx)
	Debugf("[BuildTokenValueMap %s] Building token value map, %+v", callID, rt.Config.Tokens)
	// Debug: print live call stack
	pcs := make([]uintptr, 10)
	n := runtime.Callers(2, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	Debugf("[BuildTokenValueMap %s] call stack:", callID)
	for {
		frame, more := frames.Next()
		Debugf("[BuildTokenValueMap %s]   %s\n\t%s:%d", callID, frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}
	values := make(map[string]any)
	if rt.Config == nil {
		Debugf("[BuildTokenValueMap %s] Runtime configuration is nil; returning empty token map", callID)
		return values
	}
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
func processStackMatches(stackMatches []StackMatch, frames []runtime.Frame) (string, error) {
	for _, frame := range frames {
		for _, sm := range stackMatches {
			if sm.CalledFrom == "" {
				continue
			}
			matched, err := regexp.MatchString(sm.CalledFrom, frame.Function)
			if err != nil {
				return "", fmt.Errorf("invalid regex in CalledFrom for StackMatch %q: %w", sm.Name, err)
			}
			if matched {
				return sm.Display, nil
			}
		}
	}
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
func parseKeyvals(ctx context.Context, kv ...any) map[string]any {
	callID := globalCallID(ctx)

	// Check if the length of kv is odd
	if len(kv)%2 != 0 {
		Debugf("[parseKeyvals %s] Odd number of key-vals: dropping the last element", callID)
		kv = kv[:len(kv)-1] // Remove the last element
	}
	result := make(map[string]any)
	for i := 0; i < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			Debugf("[parseKeyvals %s] Invalid key type at index %d: expected string, got %T", callID, i, kv[i])
			return map[string]any{}
		}
		result[key] = kv[i+1]
	}
	return result
}

// RenderTemplate renders a named template from the config using the provided token values.
func (cfg *Config) RenderTemplate(ctx context.Context, name string, values map[string]any) (string, error) {
	callID := globalCallID(ctx)
	Debugf("[RenderTemplate %s] Rendering template %q with values: %v", callID, name, values)
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
	vars := CollectTemplateVariables(tmpl)
	// Pre-populate missing values with fallback
	for _, v := range vars {
		if _, ok := values[v]; !ok {
			Debugf("[RenderTemplate %s] Fallback for template variable %q: not found in values", callID, v)
			values[v] = fallbackMessage(cfg, v, "template variable not found in values")
		}
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, values)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// CollectTemplateVariables walks the template AST and returns a list of all variable names referenced.
func CollectTemplateVariables(tmpl *template.Template) []string {
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

func fallbackMessage(cfg *Config, tokenName string, msg string) string {
	mode := "empty"
	if cfg != nil && cfg.Smarterr != nil && cfg.Smarterr.TokenErrorMode != nil && *cfg.Smarterr.TokenErrorMode != "" {
		mode = *cfg.Smarterr.TokenErrorMode
	}
	switch mode {
	case "detailed":
		if msg != "" {
			return fmt.Sprintf("[unresolved token: %s] (%s)", tokenName, msg)
		}
		return fmt.Sprintf("[unresolved token: %s]", tokenName)
	case "placeholder":
		return fmt.Sprintf("<%s>", tokenName)
	case "empty":
		fallthrough
	default:
		return ""
	}
}

// resolveHints processes hint suggestions for an error string, returning joined suggestions and diagnostics.
func resolveHints(ctx context.Context, errStr string, cfg *Config) string {
	callID := globalCallID(ctx)
	var suggestions []string
	matchMode := "all"
	joinChar := "\n"
	if cfg.Smarterr != nil {
		if cfg.Smarterr.HintMatchMode != nil && *cfg.Smarterr.HintMatchMode != "" {
			matchMode = *cfg.Smarterr.HintMatchMode
		}
		if cfg.Smarterr.HintJoinChar != nil {
			joinChar = *cfg.Smarterr.HintJoinChar
		}
	}
	for _, hint := range cfg.Hints {
		Debugf("[resolveHints %s] Checking hint %q against error: %s", callID, hint.Name, errStr)
		matched := true
		if hint.ErrorContains != nil && *hint.ErrorContains != "" {
			if !strings.Contains(errStr, *hint.ErrorContains) {
				Debugf("[resolveHints %s] Hint %q did not match error_contains: %s", callID, hint.Name, *hint.ErrorContains)
				matched = false
			} else {
				Debugf("[resolveHints %s] Hint %q matched error_contains: %s", callID, hint.Name, *hint.ErrorContains)
			}
		}
		if hint.RegexMatch != nil && *hint.RegexMatch != "" {
			re, err := regexp.Compile(*hint.RegexMatch)
			if err != nil {
				Debugf("[resolveHints %s] Hint %q regex compile error: %v", callID, hint.Name, err)
				matched = false
			} else if !re.MatchString(errStr) {
				Debugf("[resolveHints %s] Hint %q did not match regex: %s", callID, hint.Name, *hint.RegexMatch)
				matched = false
			} else {
				Debugf("[resolveHints %s] Hint %q matched regex: %s", callID, hint.Name, *hint.RegexMatch)
			}
		}
		if matched {
			suggestions = append(suggestions, hint.Suggestion)
			if matchMode == "first" {
				break
			}
		}
		if matchMode == "first" && len(suggestions) > 0 {
			break
		}
	}
	return strings.Join(suggestions, joinChar)
}
