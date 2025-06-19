package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/YakDriver/smarterr"
	"github.com/YakDriver/smarterr/internal"
	"github.com/spf13/cobra"
)

var quietFlag bool
var silentFlag bool

func init() {
	validateCmd.Flags().StringVarP(&startDir, "start-dir", "d", "", "Directory where code using smarterr lives (default: current directory). This is typically where the error occurs.")
	validateCmd.Flags().StringVarP(&baseDir, "base-dir", "b", "", "Parent directory where go:embed is used (optional, but recommended for proper config layering as in the application). If not set, config applies only to the current directory.")
	validateCmd.Flags().BoolVarP(&debugFlag, "debug", "D", false, "Enable smarterr debug output (even if config fails to load)")
	validateCmd.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "Only output errors (suppresses merged config and warnings)")
	validateCmd.Flags().BoolVarP(&silentFlag, "silent", "S", false, "No output, only exit code (non-zero if errors)")
	rootCmd.AddCommand(validateCmd)
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate smarterr configuration for a directory",
	Long:  `Validate the merged smarterr configuration for a directory. Checks for parse errors and config loading issues.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if debugFlag {
			internal.EnableDebugForce()
		}
		if baseDir == "" {
			fmt.Println("WARNING: --base-dir is not set. Config will only apply to the current directory. For proper config layering, set --base-dir to the directory where go:embed is used in your application.")
		}
		// Ensure baseDir and startDir are absolute
		absBaseDir, err := filepath.Abs(baseDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute baseDir: %w", err)
		}
		absStartDir := startDir
		if absStartDir == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			absStartDir = cwd
		}
		absStartDir, err = filepath.Abs(absStartDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute startDir: %w", err)
		}

		if !silentFlag && !quietFlag {
			fmt.Printf("Validating configuration...\nStart dir: %s\nBase dir: %s\n", absStartDir, absBaseDir)
		}

		// Compute relative path from baseDir to startDir
		relStartDir, err := filepath.Rel(absBaseDir, absStartDir)
		if err != nil {
			return fmt.Errorf("failed to relativize startDir: %w", err)
		}
		if strings.HasPrefix(relStartDir, "..") {
			return fmt.Errorf("startDir must be inside baseDir")
		}

		// Use a real FS rooted at baseDir
		fsys := smarterr.NewWrappedFS(absBaseDir)

		// Pass the relative stack path
		relStackPaths := []string{relStartDir}
		cfg, err := internal.LoadConfig(context.Background(), fsys, relStackPaths, ".")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Config load error: %v\n", err)
			return fmt.Errorf("config validation failed")
		}

		var allErrs []error
		var allWarnings []string

		// --- Smarterr block validation ---
		errs, warnings := validateSmarterrBlock(cfg)
		allErrs = append(allErrs, errs...)
		allWarnings = append(allWarnings, warnings...)

		// --- Template name validation ---
		errs, warnings = validateTemplateNames(cfg)
		allErrs = append(allErrs, errs...)
		allWarnings = append(allWarnings, warnings...)

		// --- Template vars and tokens validation ---
		errs, warnings = validateTemplateVarsAndTokens(cfg)
		allErrs = append(allErrs, errs...)
		allWarnings = append(allWarnings, warnings...)

		// --- Token fields validation ---
		errs, warnings = validateTokenFields(cfg)
		allErrs = append(allErrs, errs...)
		allWarnings = append(allWarnings, warnings...)

		// --- Token transforms validation ---
		errs, warnings = validateTokenTransforms(cfg)
		allErrs = append(allErrs, errs...)
		allWarnings = append(allWarnings, warnings...)

		// --- Stack matches validation ---
		errs, warnings = validateStackMatches(cfg)
		allErrs = append(allErrs, errs...)
		allWarnings = append(allWarnings, warnings...)

		// --- Transform steps validation ---
		errs, warnings = validateTransformSteps(cfg)
		allErrs = append(allErrs, errs...)
		allWarnings = append(allWarnings, warnings...)

		if !silentFlag && !quietFlag {
			fmt.Println("Merged config:")
			// Convert the configuration to HCL format
			hclBytes, err := convertConfigToHCL(cfg)
			if err != nil {
				return fmt.Errorf("failed to convert config to HCL: %w", err)
			}
			// Output the configuration
			fmt.Println(string(hclBytes))
		}

		// Print warnings and errors
		if !silentFlag && !quietFlag && len(allWarnings) > 0 {
			fmt.Println("\nWarnings:")
			for _, w := range allWarnings {
				fmt.Printf("  - %s\n", w)
			}
		}
		if len(allErrs) > 0 {
			if !silentFlag {
				fmt.Println("\nErrors:")
				for _, e := range allErrs {
					fmt.Printf("  - %s\n", e)
				}
				return fmt.Errorf("config validation failed (%d error(s))", len(allErrs))
			}
			// silentFlag: exit non-zero, but no output
			return fmt.Errorf("")
		}

		if !silentFlag && !quietFlag {
			fmt.Println("Config loaded and validated successfully.")
		}
		return nil
	},
}

// Canonical template names (should match smarterr.go)
var canonicalTemplateNames = []string{
	smarterr.DiagnosticSummaryKey,
	smarterr.DiagnosticDetailKey,
	smarterr.ErrorSummaryKey,
	smarterr.ErrorDetailKey,
	smarterr.LogErrorKey,
	smarterr.LogWarnKey,
	smarterr.LogInfoKey,
}

// validateTemplateNames checks that all template names are canonical and warns if any canonical is missing.
func validateTemplateNames(cfg *internal.Config) (errs []error, warnings []string) {
	templateNames := make(map[string]struct{})
	for _, tmpl := range cfg.Templates {
		templateNames[tmpl.Name] = struct{}{}
		found := false
		for _, canonical := range canonicalTemplateNames {
			if tmpl.Name == canonical {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Errorf("template %q is not a recognized canonical template name", tmpl.Name))
		}
	}
	// Warn if any canonical template is missing
	for _, canonical := range canonicalTemplateNames {
		if _, ok := templateNames[canonical]; !ok {
			warnings = append(warnings, fmt.Sprintf("template %q is not defined", canonical))
		}
	}
	return
}

// validateTemplateVarsAndTokens checks for template vars without tokens (error) and tokens unused in templates (warning).
func validateTemplateVarsAndTokens(cfg *internal.Config) (errs []error, warnings []string) {
	tokenNames := make(map[string]struct{})
	for _, t := range cfg.Tokens {
		tokenNames[t.Name] = struct{}{}
	}

	// Collect all template variables used in all templates
	templateVars := make(map[string]struct{})
	for _, tmpl := range cfg.Templates {
		t, err := template.New(tmpl.Name).Parse(tmpl.Format)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse template %q: %v", tmpl.Name, err))
			continue
		}
		vars := internal.CollectTemplateVariables(t)
		for _, v := range vars {
			templateVars[v] = struct{}{}
		}
	}

	// Error: template var exists that doesn't correspond to a token
	for v := range templateVars {
		if _, ok := tokenNames[v]; !ok {
			errs = append(errs, fmt.Errorf("template variable %q is used in a template but no token with that name exists", v))
		}
	}

	// Warning: token exists that's not used in a template
	for t := range tokenNames {
		if _, ok := templateVars[t]; !ok {
			warnings = append(warnings, fmt.Sprintf("token %q is defined but not used in any template", t))
		}
	}
	return
}

// validateStackMatches checks that all stack_matches referenced by tokens exist, and warns if any stack_match is unused.
func validateStackMatches(cfg *internal.Config) (errs []error, warnings []string) {
	// Collect all defined stack_match names
	defined := make(map[string]struct{})
	for _, sm := range cfg.StackMatches {
		defined[sm.Name] = struct{}{}
	}
	// Track usage of stack_matches
	used := make(map[string]struct{})
	for _, t := range cfg.Tokens {
		for _, smName := range t.StackMatches {
			if _, ok := defined[smName]; !ok {
				errs = append(errs, fmt.Errorf("token %q references undefined stack_match %q", t.Name, smName))
			} else {
				used[smName] = struct{}{}
			}
		}
	}
	// Warn if any stack_match is not used
	for smName := range defined {
		if _, ok := used[smName]; !ok {
			warnings = append(warnings, fmt.Sprintf("stack_match %q is defined but not used in any token's stack_matches", smName))
		}
	}
	return
}

// validateTokenTransforms checks that all token transforms exist, and warns if any transform is unused.
func validateTokenTransforms(cfg *internal.Config) (errs []error, warnings []string) {
	// Collect all defined transform names
	defined := make(map[string]struct{})
	for _, tr := range cfg.Transforms {
		defined[tr.Name] = struct{}{}
	}
	// Track usage of transforms
	used := make(map[string]struct{})
	for _, t := range cfg.Tokens {
		for _, trName := range t.Transforms {
			if _, ok := defined[trName]; !ok {
				errs = append(errs, fmt.Errorf("token %q references undefined transform %q", t.Name, trName))
			} else {
				used[trName] = struct{}{}
			}
		}
		// Also check field_transforms
		for _, trNames := range t.FieldTransforms {
			for _, trName := range trNames {
				if _, ok := defined[trName]; !ok {
					errs = append(errs, fmt.Errorf("token %q field_transforms references undefined transform %q", t.Name, trName))
				} else {
					used[trName] = struct{}{}
				}
			}
		}
	}
	// Warn if any transform is not used
	for trName := range defined {
		if _, ok := used[trName]; !ok {
			warnings = append(warnings, fmt.Sprintf("transform %q is defined but not used by any token", trName))
		}
	}
	return
}

// validateTransformSteps checks that all steps referenced by transforms exist, and warns if any step is unused.
func validateTransformSteps(cfg *internal.Config) (errs []error, warnings []string) {
	// Supported step types
	supported := map[string]struct{}{
		"strip_prefix": {},
		"strip_suffix": {},
		"remove":       {},
		"replace":      {},
		"trim_space":   {},
		"fix_space":    {},
		"lower":        {},
		"upper":        {},
	}
	used := make(map[string]struct{})
	for _, tr := range cfg.Transforms {
		for i, step := range tr.Steps {
			if _, ok := supported[step.Type]; !ok {
				errs = append(errs, fmt.Errorf("transform %q has step with undefined type %q", tr.Name, step.Type))
				continue
			} else {
				used[step.Type] = struct{}{}
			}

			// Validation per step type
			switch step.Type {
			case "strip_prefix", "strip_suffix", "remove":
				hasValue := step.Value != nil && *step.Value != ""
				hasRegex := step.Regex != nil && *step.Regex != ""
				if !hasValue && !hasRegex {
					errs = append(errs, fmt.Errorf("transform %q step %d (%q) must have either 'value' or 'regex' set", tr.Name, i, step.Type))
				}
				if hasValue && hasRegex {
					errs = append(errs, fmt.Errorf("transform %q step %d (%q) cannot have both 'value' and 'regex' set", tr.Name, i, step.Type))
				}
			case "replace":
				hasValue := step.Value != nil && *step.Value != ""
				hasRegex := step.Regex != nil && *step.Regex != ""
				hasWith := step.With != nil && *step.With != ""
				if !hasWith {
					errs = append(errs, fmt.Errorf("transform %q step %d (replace) must have 'with' set", tr.Name, i))
				}
				if !hasValue && !hasRegex {
					errs = append(errs, fmt.Errorf("transform %q step %d (replace) must have either 'value' or 'regex' set", tr.Name, i))
				}
				if hasValue && hasRegex {
					errs = append(errs, fmt.Errorf("transform %q step %d (replace) cannot have both 'value' and 'regex' set", tr.Name, i))
				}
			case "trim_space", "fix_space", "lower", "upper":
				if step.Value != nil {
					warnings = append(warnings, fmt.Sprintf("transform %q step %d (%q) should not have 'value' set (will be ignored)", tr.Name, i, step.Type))
				}
				if step.Regex != nil {
					warnings = append(warnings, fmt.Sprintf("transform %q step %d (%q) should not have 'regex' set (will be ignored)", tr.Name, i, step.Type))
				}
				if step.With != nil {
					warnings = append(warnings, fmt.Sprintf("transform %q step %d (%q) should not have 'with' set (will be ignored)", tr.Name, i, step.Type))
				}
			}

			// If step has a regex, try to compile it
			if step.Regex != nil {
				if _, err := regexp.Compile(*step.Regex); err != nil {
					errs = append(errs, fmt.Errorf("transform %q step %d (type %q) has invalid regex: %v", tr.Name, i, step.Type, err))
				}
			}
		}
	}
	return
}

// validateSmarterrBlock checks smarterr block fields for valid values.
func validateSmarterrBlock(cfg *internal.Config) (errs []error, warnings []string) {
	if cfg.Smarterr == nil {
		return
	}
	if cfg.Smarterr.TokenErrorMode != nil {
		mode := *cfg.Smarterr.TokenErrorMode
		if mode != "detailed" && mode != "placeholder" && mode != "empty" {
			errs = append(errs, fmt.Errorf("smarterr.token_error_mode must be one of 'detailed', 'placeholder', or 'empty' (got %q)", mode))
		}
	}
	if cfg.Smarterr.HintJoinChar != nil {
		if len(*cfg.Smarterr.HintJoinChar) > 2 {
			warnings = append(warnings, fmt.Sprintf("smarterr.hint_join_char is set to %q (longer than 2 characters)", *cfg.Smarterr.HintJoinChar))
		}
	}
	if cfg.Smarterr.HintMatchMode != nil {
		mode := *cfg.Smarterr.HintMatchMode
		if mode != "all" && mode != "first" {
			errs = append(errs, fmt.Errorf("smarterr.hint_match_mode must be 'all' or 'first' (got %q)", mode))
		}
	}
	return
}

// validateTokenFields checks for misconfiguration, missing, or conflicting fields in tokens.
func validateTokenFields(cfg *internal.Config) (errs []error, warnings []string) {
	for _, t := range cfg.Tokens {
		source := t.Source
		set := func(s *string) bool { return s != nil && *s != "" }
		countSet := 0
		if set(t.Parameter) {
			countSet++
		}
		if set(t.Context) {
			countSet++
		}
		if set(t.Arg) {
			countSet++
		}
		if len(t.StackMatches) > 0 {
			countSet++
		}

		// If source is not set, infer as in Resolve
		inferredSource := source
		if source == "" {
			switch {
			case set(t.Parameter):
				inferredSource = "parameter"
			case set(t.Context):
				inferredSource = "context"
			case set(t.Arg):
				inferredSource = "arg"
			case len(t.StackMatches) > 0:
				inferredSource = "call_stack"
			default:
				inferredSource = "parameter"
			}
			if countSet > 1 {
				errs = append(errs, fmt.Errorf("token %q: multiple fields set (parameter, context, arg, stack_matches) with no source; this is ambiguous", t.Name))
			}
		}

		// Now validate based on (inferred) source
		switch inferredSource {
		case "parameter":
			if !set(t.Parameter) {
				errs = append(errs, fmt.Errorf("token %q: source=parameter but 'parameter' field is not set", t.Name))
			}
			if set(t.Context) || set(t.Arg) || len(t.StackMatches) > 0 {
				warnings = append(warnings, fmt.Sprintf("token %q: source=parameter should not set context, arg, or stack_matches", t.Name))
			}
		case "context":
			if !set(t.Context) {
				errs = append(errs, fmt.Errorf("token %q: source=context but 'context' field is not set", t.Name))
			}
			if set(t.Parameter) || set(t.Arg) || len(t.StackMatches) > 0 {
				warnings = append(warnings, fmt.Sprintf("token %q: source=context should not set parameter, arg, or stack_matches", t.Name))
			}
		case "arg":
			if !set(t.Arg) {
				errs = append(errs, fmt.Errorf("token %q: source=arg but 'arg' field is not set", t.Name))
			}
			if set(t.Parameter) || set(t.Context) || len(t.StackMatches) > 0 {
				warnings = append(warnings, fmt.Sprintf("token %q: source=arg should not set parameter, context, or stack_matches", t.Name))
			}
		case "call_stack", "error_stack":
			if len(t.StackMatches) == 0 {
				errs = append(errs, fmt.Errorf("token %q: source=%s but stack_matches is not set", t.Name, inferredSource))
			}
			if set(t.Parameter) || set(t.Context) || set(t.Arg) {
				warnings = append(warnings, fmt.Sprintf("token %q: source=%s should not set parameter, context, or arg", t.Name, inferredSource))
			}
		case "diagnostic":
			if set(t.Parameter) || set(t.Context) || set(t.Arg) || len(t.StackMatches) > 0 {
				warnings = append(warnings, fmt.Sprintf("token %q: source=diagnostic should not set parameter, context, arg, or stack_matches", t.Name))
			}
		case "hints", "error":
			if set(t.Parameter) || set(t.Context) || set(t.Arg) || len(t.StackMatches) > 0 {
				warnings = append(warnings, fmt.Sprintf("token %q: source=%s should not set parameter, context, arg, or stack_matches", t.Name, inferredSource))
			}
		}
		// If stack_matches is set but source is not call_stack or error_stack, warn
		if len(t.StackMatches) > 0 && inferredSource != "call_stack" && inferredSource != "error_stack" {
			warnings = append(warnings, fmt.Sprintf("token %q: stack_matches is set but source is not call_stack or error_stack (actual: %s)", t.Name, inferredSource))
		}
	}
	return
}
