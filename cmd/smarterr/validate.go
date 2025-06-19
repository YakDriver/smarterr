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
	validateCmd.Flags().StringVar(&startDir, "start-dir", "", "Starting directory for configuration traversal (default is current directory)")
	validateCmd.Flags().StringVar(&baseDir, "base-dir", "", "Base directory to restrict traversal (optional)")
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

		fmt.Println("Merged config:")
		// Convert the configuration to HCL format
		hclBytes, err := convertConfigToHCL(cfg)
		if err != nil {
			return fmt.Errorf("failed to convert config to HCL: %w", err)
		}

		// Output the configuration
		fmt.Println(string(hclBytes))

		fmt.Println("Config loaded successfully.")
		return nil
	},
}
