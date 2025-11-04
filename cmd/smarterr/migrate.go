package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/YakDriver/smarterr/internal/migrate"
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

		if !isGoFile(path) {
			return nil
		}

		return migrateFile(path)
	})
}

func isGoFile(path string) bool {
	return strings.HasSuffix(path, ".go") &&
		!strings.HasSuffix(path, "_test.go") &&
		!strings.Contains(path, "_gen")
}

func migrateFile(filename string) error {
	if verboseFlag {
		fmt.Printf("Processing: %s\n", filename)
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filename, err)
	}

	if err := validateGoSyntax(filename, content); err != nil {
		return err
	}

	if !migrate.NeedsMigration(string(content)) {
		if verboseFlag {
			fmt.Printf("Skipped: %s (no migration needed)\n", filename)
		}
		return nil
	}

	migrator := migrate.NewMigrator(migrate.MigratorOptions{
		DryRun:  dryRunFlag,
		Verbose: verboseFlag,
	})

	migratedContent := migrator.MigrateContent(string(content))

	if migratedContent == string(content) {
		if verboseFlag {
			fmt.Printf("Skipped: %s (no changes)\n", filename)
		}
		return nil
	}

	if err := validateGoSyntax(filename, []byte(migratedContent)); err != nil {
		if debugFlag {
			fmt.Printf("=== MIGRATED CONTENT WITH SYNTAX ERRORS ===\n")
			fmt.Printf("File: %s\n", filename)
			fmt.Printf("Content:\n%s\n", migratedContent)
			fmt.Printf("=== END MIGRATED CONTENT ===\n")
		}
		return fmt.Errorf("migration resulted in invalid Go code: %w", err)
	}

	if dryRunFlag {
		fmt.Printf("Would migrate: %s\n", filename)
		return nil
	}

	if err := writeFile(filename, migratedContent); err != nil {
		return err
	}

	if err := formatFile(filename); err != nil {
		fmt.Printf("Warning: formatting failed for %s: %v\n", filename, err)
	}

	fmt.Printf("Migrated: %s\n", filename)
	return nil
}

func validateGoSyntax(filename string, content []byte) error {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", filename, err)
	}
	return nil
}

func writeFile(filename, content string) error {
	info, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("getting file info for %s: %w", filename, err)
	}

	if err := os.WriteFile(filename, []byte(content), info.Mode()); err != nil {
		return fmt.Errorf("writing %s: %w", filename, err)
	}
	return nil
}

func formatFile(filename string) error {
	if err := exec.Command("goimports", "-w", filename).Run(); err != nil {
		return fmt.Errorf("goimports: %w", err)
	}
	if err := exec.Command("gofmt", "-w", filename).Run(); err != nil {
		return fmt.Errorf("gofmt: %w", err)
	}
	return nil
}
