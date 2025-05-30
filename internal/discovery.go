// discovery.go
// Config discovery and loading logic for smarterr
package internal

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/YakDriver/smarterr/filesystem"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// LoadConfig loads and merges configuration files from a filesystem.
func LoadConfig(fsys filesystem.FileSystem, relStackPaths []string, baseDir string) (*Config, error) {
	return loadConfigMultiStack(fsys, relStackPaths, baseDir)
}

// loadConfigMultiStack is the internal implementation for loading and merging config files
// based on multiple stack paths. This is optimized for embedded FS, but can be adapted for
// real FS in the future.
func loadConfigMultiStack(fsys filesystem.FileSystem, relStackPaths []string, baseDir string) (*Config, error) {
	configs, err := collectConfigsForStack(fsys, relStackPaths, baseDir)
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return &Config{}, nil
	}
	return mergeConfigs(configs), nil
}

// collectConfigsForStack collects and loads all config files relevant to the provided stack paths.
// This is the main entry for config discovery in embedded FS mode.
func collectConfigsForStack(fsys filesystem.FileSystem, relStackPaths []string, baseDir string) ([]*Config, error) {
	var configs []*Config
	globalConfigPath, candidateConfigs, err := findAllConfigPaths(fsys)
	if err != nil {
		return nil, err
	}

	// Always include the global config if present
	if globalConfigPath != "" {
		cfg, err := loadConfigFile(fsys, globalConfigPath)
		if err != nil {
			return nil, fmt.Errorf("error loading global config: %w", err)
		}
		configs = append(configs, cfg)
	}

	sep := string(filepath.Separator)
	for _, configPath := range candidateConfigs {
		configDir := filepath.Dir(configPath)
		needle := baseDir + sep + configDir
		for _, stackPath := range relStackPaths {
			if strings.Contains(stackPath, needle) {
				cfg, err := loadConfigFile(fsys, configPath)
				if err != nil {
					return nil, fmt.Errorf("error loading config %s: %w", configPath, err)
				}
				configs = append(configs, cfg)
				break // Only need to match once per config
			}
		}
	}
	return configs, nil
}

// findAllConfigPaths scans the FS for all smarterr.hcl files, returning the global config path and other candidates.
func findAllConfigPaths(fsys filesystem.FileSystem) (globalConfig string, candidateConfigs []string, err error) {
	err = fsys.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ConfigFileName) {
			return nil
		}
		if strings.HasPrefix(path, "smarterr/") {
			globalConfig = path
		} else {
			candidateConfigs = append(candidateConfigs, path)
		}
		return nil
	})
	return
}

// loadConfigFile loads a single config file from the FS and parses it into a Config struct.
func loadConfigFile(fsys filesystem.FileSystem, path string) (*Config, error) {
	parser := hclparse.NewParser()
	fileBytes, err := fsys.ReadFile(path)
	if err != nil {
		return nil, err
	}
	file, diags := parser.ParseHCL(fileBytes, path)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse error: %s", diags.Error())
	}
	var partial Config
	decodeDiags := gohcl.DecodeBody(file.Body, nil, &partial)
	if decodeDiags.HasErrors() {
		return nil, fmt.Errorf("decode error: %s", decodeDiags.Error())
	}
	return &partial, nil
}
