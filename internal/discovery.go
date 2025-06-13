// discovery.go
// Config discovery and loading logic for smarterr
package internal

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// LoadConfig loads and merges configuration files from a filesystem.
func LoadConfig(fsys FileSystem, relStackPaths []string, baseDir string) (*Config, error) {
	Debugf("LoadConfig called with baseDir=%q relStackPaths=%v", baseDir, relStackPaths)
	return loadConfigMultiStack(fsys, relStackPaths, baseDir)
}

// LoadConfigWithDiagnostics loads and merges configuration files from a filesystem, collecting diagnostics.
func LoadConfigWithDiagnostics(fsys FileSystem, relStackPaths []string, baseDir string, diagnostics *[]error) (*Config, error) {
	cfg, err := LoadConfig(fsys, relStackPaths, baseDir)
	if err != nil {
		if diagnostics != nil {
			*diagnostics = append(*diagnostics, err)
		}
	}
	return cfg, err
}

// loadConfigMultiStack is the internal implementation for loading and merging config files
// based on multiple stack paths. This is optimized for embedded FS, but can be adapted for
// real FS in the future.
func loadConfigMultiStack(fsys FileSystem, relStackPaths []string, baseDir string) (*Config, error) {
	configs, err := collectConfigsForStack(fsys, relStackPaths, baseDir)
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return &Config{}, nil
	}
	merged := mergeConfigs(configs)
	EnableDebug(merged) // Enable internal debug output based on config
	return merged, nil
}

// collectConfigsForStack collects and loads all config files relevant to the provided stack paths.
// This is the main entry for config discovery in embedded FS mode.
func collectConfigsForStack(fsys FileSystem, relStackPaths []string, baseDir string) ([]*Config, error) {
	type configWithPath struct {
		cfg  *Config
		path string
	}
	var cfgsWithPaths []configWithPath
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
		cfgsWithPaths = append(cfgsWithPaths, configWithPath{cfg, globalConfigPath})
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
				cfgsWithPaths = append(cfgsWithPaths, configWithPath{cfg, configPath})
				break // Only need to match once per config
			}
		}
	}
	// Sort by path depth (least specific first, most specific last)
	sort.Slice(cfgsWithPaths, func(i, j int) bool {
		return strings.Count(cfgsWithPaths[i].path, sep) < strings.Count(cfgsWithPaths[j].path, sep)
	})
	var configs []*Config
	for _, c := range cfgsWithPaths {
		configs = append(configs, c.cfg)
	}
	return configs, nil
}

// findAllConfigPaths scans the FS for all smarterr.hcl files, returning the global config path and other candidates.
func findAllConfigPaths(fsys FileSystem) (globalConfig string, candidateConfigs []string, err error) {
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
func loadConfigFile(fsys FileSystem, path string) (*Config, error) {
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

// FileSystem defines an interface for filesystem operations, including file existence checks.
type FileSystem interface {
	Open(name string) (fs.File, error)
	ReadFile(name string) ([]byte, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
	Exists(name string) bool
}

// WrappedFS implements FileSystem for a generic fs.FS.
type WrappedFS struct {
	FS fs.FS
}

func NewWrappedFS(root string) *WrappedFS {
	return &WrappedFS{
		FS: os.DirFS(root),
	}
}

func (d *WrappedFS) Open(name string) (fs.File, error) {
	return d.FS.Open(name)
}

func (d *WrappedFS) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(d.FS, name)
}

func (d *WrappedFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(d.FS, root, fn)
}

// Exists checks if a file exists in the wrapped filesystem.
func (d *WrappedFS) Exists(path string) bool {
	f, err := d.FS.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	stat, err := f.Stat()
	return err == nil && !stat.IsDir()
}
