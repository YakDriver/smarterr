package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/YakDriver/smarterr"
	"github.com/YakDriver/smarterr/internal"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/spf13/cobra"
	"github.com/zclconf/go-cty/cty"
)

var startDir string
var baseDir string

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show the effective smarterr configuration for a directory",
	Long: `This command prints the merged smarterr configuration that would apply
at the specified directory path. It helps debug layered config resolution.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default startDir to current working directory
		if startDir == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			startDir = cwd
		}

		// Ensure baseDir is absolute
		if baseDir == "" {
			return fmt.Errorf("--base-dir is required")
		}

		fmt.Printf("Loading configuration...\nStart dir: %s\nBase dir: %s\n", startDir, baseDir)

		// Create FileSystem rooted at baseDir
		fsys := smarterr.NewWrappedFS(baseDir)

		// Compute paths relative to baseDir
		relStartDir, err := filepath.Rel(baseDir, startDir)
		if err != nil {
			return fmt.Errorf("failed to relativize startDir: %w", err)
		}

		// Load config (pass relStackPaths as []string)
		cfg, err := internal.LoadConfig(fsys, []string{relStartDir}, ".")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Printf("Raw merged config: %+v\n", cfg)

		fmt.Println("Merged config:")
		// Convert the configuration to HCL format
		hclBytes, err := convertConfigToHCL(cfg)
		if err != nil {
			return fmt.Errorf("failed to convert config to HCL: %w", err)
		}

		// Output the configuration
		fmt.Println(string(hclBytes))
		return nil
	},
}

func init() {
	configCmd.Flags().StringVar(&startDir, "start-dir", "", "Starting directory for configuration traversal (default is current directory)")
	configCmd.Flags().StringVar(&baseDir, "base-dir", "", "Base directory to restrict traversal (optional)")
	rootCmd.AddCommand(configCmd)
}

func convertConfigToHCL(cfg *internal.Config) ([]byte, error) {
	file := hclwrite.NewEmptyFile()
	body := file.Body()

	// Top-level attributes
	if cfg.TokenErrorMode != "" {
		body.SetAttributeValue("token_error_mode", cty.StringVal(cfg.TokenErrorMode))
	}

	// Tokens
	for _, token := range cfg.Tokens {
		block := body.AppendNewBlock("token", []string{token.Name})
		b := block.Body()
		// b.SetAttributeValue("position", cty.NumberIntVal(int64(token.Position))) // No Position field
		b.SetAttributeValue("source", cty.StringVal(token.Source))

		if token.Parameter != nil {
			b.SetAttributeValue("parameter", cty.StringVal(*token.Parameter))
		}
		if token.Arg != nil {
			b.SetAttributeValue("arg", cty.StringVal(*token.Arg))
		}
		if token.Context != nil {
			b.SetAttributeValue("context", cty.StringVal(*token.Context))
		}
		if token.Pattern != nil {
			b.SetAttributeValue("pattern", cty.StringVal(*token.Pattern))
		}
		if token.Replace != nil {
			b.SetAttributeValue("replace", cty.StringVal(*token.Replace))
		}
		if len(token.Transforms) > 0 {
			vals := make([]cty.Value, len(token.Transforms))
			for i, v := range token.Transforms {
				vals[i] = cty.StringVal(v)
			}
			b.SetAttributeValue("transforms", cty.ListVal(vals))
		}
		if len(token.StackMatches) > 0 {
			vals := make([]cty.Value, len(token.StackMatches))
			for i, v := range token.StackMatches {
				vals[i] = cty.StringVal(v)
			}
			b.SetAttributeValue("stack_matches", cty.ListVal(vals))
		}
	}

	// Parameters
	for _, param := range cfg.Parameters {
		block := body.AppendNewBlock("parameter", []string{param.Name})
		block.Body().SetAttributeValue("value", cty.StringVal(param.Value))
	}

	// Hints
	for _, hint := range cfg.Hints {
		block := body.AppendNewBlock("hint", []string{hint.Name})
		b := block.Body()
		if hint.ErrorContains != nil {
			b.SetAttributeValue("error_contains", cty.StringVal(*hint.ErrorContains))
		}
		if hint.RegexMatch != nil {
			b.SetAttributeValue("regex_match", cty.StringVal(*hint.RegexMatch))
		}
		b.SetAttributeValue("suggestion", cty.StringVal(hint.Suggestion))
	}

	// StackMatches
	for _, sm := range cfg.StackMatches {
		block := body.AppendNewBlock("stack_match", []string{sm.Name})
		b := block.Body()
		if sm.CalledFrom != "" {
			b.SetAttributeValue("called_from", cty.StringVal(sm.CalledFrom))
		}
		if sm.CalledAfter != "" {
			b.SetAttributeValue("called_after", cty.StringVal(sm.CalledAfter))
		}
		b.SetAttributeValue("display", cty.StringVal(sm.Display))
	}

	// SmarterrDebug block
	if cfg.SmarterrDebug != nil {
		block := body.AppendNewBlock("smarterr_debug", nil)
		b := block.Body()
		if cfg.SmarterrDebug.Output != "" {
			b.SetAttributeValue("output", cty.StringVal(cfg.SmarterrDebug.Output))
		}
	}

	return file.Bytes(), nil
}
