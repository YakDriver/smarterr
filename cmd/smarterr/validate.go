package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/YakDriver/smarterr"
	"github.com/YakDriver/smarterr/internal"
	"github.com/spf13/cobra"
)

func init() {
	validateCmd.Flags().StringVar(&startDir, "start-dir", "", "Directory where code using smarterr lives (default: current directory). This is typically where the error occurs.")
	validateCmd.Flags().StringVar(&baseDir, "base-dir", "", "Parent directory where go:embed is used (optional, but recommended for proper config layering as in the application). If not set, config applies only to the current directory.")
	validateCmd.Flags().BoolVar(&debugFlag, "debug", false, "Enable smarterr debug output (even if config fails to load)")
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

		fmt.Printf("Validating configuration...\nStart dir: %s\nBase dir: %s\n", absStartDir, absBaseDir)

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

		// --- Template name validation ---
		errs, warnings := validateTemplateNames(cfg)
		allErrs = append(allErrs, errs...)
		allWarnings = append(allWarnings, warnings...)

		fmt.Println("Merged config:")
		// Convert the configuration to HCL format
		hclBytes, err := convertConfigToHCL(cfg)
		if err != nil {
			return fmt.Errorf("failed to convert config to HCL: %w", err)
		}

		// Output the configuration
		fmt.Println(string(hclBytes))

		// Print warnings and errors
		if len(allWarnings) > 0 {
			fmt.Println("\nWarnings:")
			for _, w := range allWarnings {
				fmt.Printf("  - %s\n", w)
			}
		}
		if len(allErrs) > 0 {
			fmt.Println("\nErrors:")
			for _, e := range allErrs {
				fmt.Printf("  - %s\n", e)
			}
			return fmt.Errorf("config validation failed (%d error(s))", len(allErrs))
		}

		fmt.Println("Config loaded and validated successfully.")
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
