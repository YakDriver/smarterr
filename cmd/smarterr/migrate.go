package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var dryRunFlag bool
var verboseFlag bool

func init() {
	migrateCmd.Flags().BoolVarP(&dryRunFlag, "dry-run", "n", false, "Show what would be changed without making changes")
	migrateCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Show detailed output")
	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate [path]",
	Short: "Migrate Go code to use smarterr patterns",
	Long: `Migrate Go code to use smarterr patterns by applying automatic transformations.
This command will:
- Add required smarterr imports
- Replace legacy error handling patterns with smarterr equivalents
- Transform bare error returns to use smarterr.NewError()
- Convert diagnostic patterns to use smerr helpers

Example:
  smarterr migrate ./internal/service/myservice/`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		return migrateDirectory(path)
	},
}

func migrateDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files, test files, and generated files
		if !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") ||
			strings.Contains(path, "_gen") {
			return nil
		}

		return migrateFile(path)
	})
}

func migrateFile(filename string) error {
	if verboseFlag {
		fmt.Printf("Processing: %s\n", filename)
	}

	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filename, err)
	}

	originalContent := string(content)

	// Parse the Go file to validate syntax
	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	// Check if migration is needed
	if !needsMigration(originalContent) {
		if verboseFlag {
			fmt.Printf("  Skipped: %s (no migration needed)\n", filename)
		}
		return nil
	}

	// Apply migrations
	migratedContent := originalContent
	migratedContent = addImports(migratedContent)
	migratedContent = migratePatterns(migratedContent)

	// Check if anything changed
	if migratedContent == originalContent {
		if verboseFlag {
			fmt.Printf("  Skipped: %s (no changes)\n", filename)
		}
		return nil
	}

	// Validate the migrated code can be parsed
	_, err = parser.ParseFile(fset, filename, migratedContent, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("migration resulted in invalid Go code for %s: %w", filename, err)
	}

	if dryRunFlag {
		fmt.Printf("Would migrate: %s\n", filename)
		return nil
	}

	// Get file info for permissions
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", filename, err)
	}

	// Write the migrated content
	err = os.WriteFile(filename, []byte(migratedContent), fileInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	// Run goimports to organize imports properly
	cmd := exec.Command("goimports", "-w", filename)
	if err := cmd.Run(); err != nil {
		// Non-fatal: warn but continue
		fmt.Printf("Warning: goimports failed for %s: %v\n", filename, err)
	}

	fmt.Printf("Migrated: %s\n", filename)
	return nil
}

func needsMigration(content string) bool {
	patterns := []string{
		`response\.Diagnostics\.Append`,
		`response\.Diagnostics\.AddError`,
		`sdkdiag\.AppendFromErr`,
		`sdkdiag\.AppendErrorf`,
		`create\.AppendDiagError`,
		`create\.AddError`,
		`(?m)return nil, err$`, // Use multiline mode
		`return nil, &retry\.NotFoundError`,
		`return nil, tfresource\.NewEmptyResultError`,
		`return tfresource\.AssertSingleValueResult`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, content)
		if matched {
			return true
		}
	}
	return false
}

func addImports(content string) string {
	// Simple approach: add both imports to the end of the import block
	// and let goimports organize them properly

	// Check if imports already exist
	hasSmartErr := strings.Contains(content, `"github.com/YakDriver/smarterr"`)
	hasSmerr := strings.Contains(content, `"github.com/hashicorp/terraform-provider-aws/internal/smerr"`)

	if hasSmartErr && hasSmerr {
		return content // Both imports already present
	}

	// Find import block
	importBlockPattern := regexp.MustCompile(`(import \(\n)((?:[^\)]*\n)*?)(\))`)
	matches := importBlockPattern.FindStringSubmatch(content)
	if len(matches) != 4 {
		return content // No import block found
	}

	imports := matches[2]

	// Add smarterr import at the end if not present
	if !hasSmartErr {
		imports += "\t\"github.com/YakDriver/smarterr\"\n"
	}

	// Add smerr import at the end if not present
	if !hasSmerr {
		imports += "\t\"github.com/hashicorp/terraform-provider-aws/internal/smerr\"\n"
	}

	// Reconstruct the import block
	newContent := strings.Replace(content, matches[0], matches[1]+imports+matches[3], 1)
	return newContent
}

func migratePatterns(content string) string {
	// 1. Simple bare error returns (single line only)
	content = regexp.MustCompile(`(?m)(\s+)return nil, err$`).
		ReplaceAllString(content, `${1}return nil, smarterr.NewError(err)`)

	// 2. return nil, tfresource.NewEmptyResultError(...) (single line)
	content = regexp.MustCompile(`(?m)(\s+)return nil, tfresource\.NewEmptyResultError\(([^)]*)\)$`).
		ReplaceAllString(content, `${1}return nil, smarterr.NewError(tfresource.NewEmptyResultError($2))`)

	// 3. return tfresource.AssertSingleValueResult(...) - handle nested calls
	// First try to match the simple cases, then handle nested parentheses
	content = replaceAssertSingleValueResult(content)

	// 4. Multi-line retry.NotFoundError
	content = regexp.MustCompile(`(?m)(\s+)return nil, &retry\.NotFoundError\{\s*\n\s*LastError:\s*([^,\n]+),\s*\n\s*LastRequest:\s*([^,\n]+),?\s*\n\s*\}$`).
		ReplaceAllString(content, `${1}return nil, smarterr.NewError(&retry.NotFoundError{LastError: $2, LastRequest: $3})`)

	// 5. Single-line retry.NotFoundError
	content = regexp.MustCompile(`(?m)(\s+)return nil, &retry\.NotFoundError\{\s*LastError:\s*([^,}]+),\s*LastRequest:\s*([^,}]+),?\s*\}$`).
		ReplaceAllString(content, `${1}return nil, smarterr.NewError(&retry.NotFoundError{LastError: $2, LastRequest: $3})`)

	// 6. Framework patterns - response.Diagnostics.Append with ... variadic operator
	// Need to handle nested parentheses for calls like request.State.Get(ctx, &data)...
	content = replaceVariadicAppend(content)

	// 7. Framework patterns - response.Diagnostics.Append with fwdiag (no variadic)
	// Special case for fwdiag.NewResourceNotFoundWarningDiagnostic first (more specific)
	content = regexp.MustCompile(`(?m)(\s+)response\.Diagnostics\.Append\(fwdiag\.NewResourceNotFoundWarningDiagnostic\(([^)]+)\)\)$`).
		ReplaceAllString(content, `${1}smerr.EnrichAppendDiagnostic(ctx, &response.Diagnostics, fwdiag.NewResourceNotFoundWarningDiagnostic($2))`)

	// Handle other fwdiag patterns with nested parentheses
	content = replaceFwdiagAppend(content)

	// 8. Framework patterns - response.Diagnostics.AddError (simple single line)
	content = regexp.MustCompile(`(?m)(\s+)response\.Diagnostics\.AddError\(\s*"([^"]*)",\s*([^)]+)\.Error\(\)\s*\)$`).
		ReplaceAllString(content, `${1}smerr.AddError(ctx, &response.Diagnostics, $3)`)

	// 9. SDKv2 patterns - sdkdiag.AppendFromErr
	content = regexp.MustCompile(`sdkdiag\.AppendFromErr\(([^,]+),\s*([^)]+)\)`).
		ReplaceAllString(content, `smerr.Append(ctx, $1, $2)`)

	// 10. SDKv2 patterns - sdkdiag.AppendErrorf (more specific to avoid breaking other append calls)
	// Use [^,\n]+ to avoid matching across lines or argument boundaries, but allow parentheses
	content = regexp.MustCompile(`(?m)sdkdiag\.AppendErrorf\(([^,]+),\s*"[^"]*",\s*([^,\n]+),\s*([^,\n]+)\)$`).
		ReplaceAllString(content, `smerr.Append(ctx, $1, $3, smerr.ID, $2)`)

	content = regexp.MustCompile(`(?m)sdkdiag\.AppendErrorf\(([^,]+),\s*"[^"]*",\s*([^,\n]+)\)$`).
		ReplaceAllString(content, `smerr.Append(ctx, $1, $2)`)

	// 11. create.AppendDiagError patterns (more specific)
	content = regexp.MustCompile(`(?m)create\.AppendDiagError\(([^,]+),\s*[^)]*\)$`).
		ReplaceAllString(content, `smerr.Append(ctx, $1, err, smerr.ID, id)`)

	// 12. create.AddError patterns (more specific)
	content = regexp.MustCompile(`(?m)create\.AddError\(&([^,]+),\s*[^)]*\)$`).
		ReplaceAllString(content, `smerr.AddError(ctx, &$1, err, smerr.ID, id)`)

	return content
}

// replaceVariadicAppend handles response.Diagnostics.Append with ... variadic operator
func replaceVariadicAppend(content string) string {
	re := regexp.MustCompile(`(?m)(\s+)response\.Diagnostics\.Append\((.+)\.\.\.\)$`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the parts
		submatches := re.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}
		indent := submatches[1]
		arg := submatches[2]

		// Extract the function call before ...
		// We need to find the balanced parentheses
		return indent + "smerr.EnrichAppend(ctx, &response.Diagnostics, " + arg + ")"
	})
}

// replaceAssertSingleValueResult handles tfresource.AssertSingleValueResult with nested calls
func replaceAssertSingleValueResult(content string) string {
	re := regexp.MustCompile(`(?m)(\s+)return tfresource\.AssertSingleValueResult\((.+)\)$`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the parts
		submatches := re.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}
		indent := submatches[1]
		arg := submatches[2]

		return indent + "return smarterr.Assert(tfresource.AssertSingleValueResult(" + arg + "))"
	})
}

// replaceFwdiagAppend handles response.Diagnostics.Append with fwdiag calls that may have nested parentheses
func replaceFwdiagAppend(content string) string {
	re := regexp.MustCompile(`(?m)(\s+)response\.Diagnostics\.Append\(fwdiag\.([^(]+)\((.+)\)\)$`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the parts
		submatches := re.FindStringSubmatch(match)
		if len(submatches) != 4 {
			return match
		}
		indent := submatches[1]
		funcName := submatches[2]
		args := submatches[3]

		// Make sure we captured the full function call correctly
		// Count parentheses to ensure we got all of them
		parenCount := 0
		for _, ch := range args {
			if ch == '(' {
				parenCount++
			} else if ch == ')' {
				parenCount--
			}
		}

		// If parentheses are balanced, we got it all
		if parenCount == 0 {
			return indent + "smerr.EnrichAppendDiagnostic(ctx, &response.Diagnostics, fwdiag." + funcName + "(" + args + "))"
		}

		// Otherwise, don't replace (regex didn't capture properly)
		return match
	})
}
