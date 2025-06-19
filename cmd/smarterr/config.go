package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/YakDriver/smarterr"
	"github.com/YakDriver/smarterr/internal"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/spf13/cobra"
	"github.com/zclconf/go-cty/cty"
)

var startDir string
var baseDir string

func init() {
	configCmd.Flags().StringVar(&startDir, "start-dir", "", "Directory where code using smarterr lives (default: current directory). This is typically where the error occurs.")
	configCmd.Flags().StringVar(&baseDir, "base-dir", "", "Parent directory where go:embed is used (optional, but recommended for proper config layering as in the application). If not set, config applies only to the current directory.")
	configCmd.Flags().BoolVar(&debugFlag, "debug", false, "Enable smarterr debug output (even if config fails to load)")
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show the effective smarterr configuration for a directory",
	Long: `This command prints the merged smarterr configuration that would apply
at the specified directory path. It helps debug layered config resolution.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if debugFlag {
			fmt.Printf("Debug mode enabled\n")
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

		fmt.Printf("Loading configuration...\nStart dir: %s\nBase dir: %s\n", absStartDir, absBaseDir)

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

		// Output all config files found under baseDir
		fmt.Println("Config files found under baseDir:")
		err = filepath.Walk(absBaseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip errors
			}
			if info.IsDir() {
				return nil
			}
			if filepath.Base(path) == "smarterr.hcl" {
				rel, _ := filepath.Rel(absBaseDir, path)
				fmt.Println("  ", rel)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("error walking baseDir: %w", err)
		}

		// Pass the relative stack path
		relStackPaths := []string{relStartDir}
		cfg, err := internal.LoadConfig(context.Background(), fsys, relStackPaths, ".")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if debugFlag {
			fmt.Printf("Raw merged config: %+v\n", cfg)
		}

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

func convertConfigToHCL(cfg *internal.Config) ([]byte, error) {
	file := hclwrite.NewEmptyFile()
	body := file.Body()

	// Smarterr block (debug, token_error_mode, hint_match_mode, hint_join_char)
	if cfg.Smarterr != nil && (cfg.Smarterr.Debug || (cfg.Smarterr.TokenErrorMode != nil && *cfg.Smarterr.TokenErrorMode != "") || cfg.Smarterr.HintMatchMode != nil || cfg.Smarterr.HintJoinChar != nil) {
		smarterrBlock := body.AppendNewBlock("smarterr", nil)
		b := smarterrBlock.Body()
		if cfg.Smarterr.Debug {
			b.SetAttributeValue("debug", cty.BoolVal(true))
		}
		if cfg.Smarterr.TokenErrorMode != nil && *cfg.Smarterr.TokenErrorMode != "" {
			b.SetAttributeValue("token_error_mode", cty.StringVal(*cfg.Smarterr.TokenErrorMode))
		}
		if cfg.Smarterr.HintMatchMode != nil {
			b.SetAttributeValue("hint_match_mode", cty.StringVal(*cfg.Smarterr.HintMatchMode))
		}
		if cfg.Smarterr.HintJoinChar != nil {
			b.SetAttributeValue("hint_join_char", cty.StringVal(*cfg.Smarterr.HintJoinChar))
		}
	}

	// Tokens
	for _, token := range cfg.Tokens {
		block := body.AppendNewBlock("token", []string{token.Name})
		b := block.Body()
		if token.Source != "" {
			b.SetAttributeValue("source", cty.StringVal(token.Source))
		}
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
		if len(token.FieldTransforms) > 0 {
			ftBlock := b.AppendNewBlock("field_transforms", nil)
			ftBody := ftBlock.Body()
			for field, transforms := range token.FieldTransforms {
				vals := make([]cty.Value, len(transforms))
				for i, v := range transforms {
					vals[i] = cty.StringVal(v)
				}
				ftBody.SetAttributeValue(field, cty.ListVal(vals))
			}
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
		b.SetAttributeValue("display", cty.StringVal(sm.Display))
	}

	// Templates
	for _, tmpl := range cfg.Templates {
		block := body.AppendNewBlock("template", []string{tmpl.Name})
		block.Body().SetAttributeValue("format", cty.StringVal(tmpl.Format))
	}

	// Transforms
	for _, tr := range cfg.Transforms {
		block := body.AppendNewBlock("transform", []string{tr.Name})
		for _, step := range tr.Steps {
			stepBlock := block.Body().AppendNewBlock("step", []string{step.Type})
			b := stepBlock.Body()
			if step.Value != nil {
				b.SetAttributeValue("value", cty.StringVal(*step.Value))
			}
			if step.Regex != nil {
				b.SetAttributeValue("regex", cty.StringVal(*step.Regex))
			}
			if step.With != nil {
				b.SetAttributeValue("with", cty.StringVal(*step.With))
			}
			if step.Recurse != nil {
				b.SetAttributeValue("recurse", cty.BoolVal(*step.Recurse))
			}
		}
	}

	return file.Bytes(), nil
}
