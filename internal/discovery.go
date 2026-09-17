// discovery.go
// Config discovery and loading logic for smarterr
package internal

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// LoadConfig loads and merges configuration files from a filesystem.
func LoadConfig(ctx context.Context, fsys FileSystem, relStackPaths []string, baseDir string) (*Config, error) {
	callID := globalCallID(ctx)
	Debugf("[LoadConfig %s] called with baseDir=%q relStackPaths=%v", callID, baseDir, relStackPaths)
	return loadConfigMultiStack(ctx, fsys, relStackPaths, baseDir)
}

// loadConfigMultiStack is the internal implementation for loading and merging config files
// based on multiple stack paths. This is optimized for embedded FS, but can be adapted for
// real FS in the future.
func loadConfigMultiStack(ctx context.Context, fsys FileSystem, relStackPaths []string, baseDir string) (*Config, error) {
	callID := globalCallID(ctx)
	Debugf("[loadConfigMultiStack %s] called with baseDir=%q relStackPaths=%v", callID, baseDir, relStackPaths)
	configs, err := collectConfigsForStack(ctx, fsys, relStackPaths, baseDir)
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return &Config{}, nil
	}
	merged := mergeConfigs(ctx, configs)
	EnableDebug(merged) // Enable internal debug output based on config
	return merged, nil
}

// collectConfigsForStack collects and loads all config files relevant to the provided stack paths.
// This is the main entry for config discovery in embedded FS mode.
func collectConfigsForStack(ctx context.Context, fsys FileSystem, relStackPaths []string, baseDir string) ([]*Config, error) {
	callID := globalCallID(ctx)
	Debugf("[collectConfigsForStack %s] called with baseDir=%q relStackPaths=%v", callID, baseDir, relStackPaths)
	// Find all config files in
	type configWithPath struct {
		cfg    *Config
		path   string
		global bool
	}
	var cfgsWithPaths []configWithPath
	globalConfigPath, candidateConfigs, err := findAllConfigPaths(ctx, fsys)
	if err != nil {
		return nil, err
	}

	// Always include the global config if present
	if globalConfigPath != "" {
		cfg, err := loadConfigFile(ctx, fsys, globalConfigPath)
		if err != nil {
			return nil, fmt.Errorf("error loading global config: %w", err)
		}
		cfgsWithPaths = append(cfgsWithPaths, configWithPath{cfg, globalConfigPath, true})
	}

	// io/fs paths (config paths and the embedded FS) always use "/", and frame
	// paths are normalized to "/" upstream, so match on "/" regardless of host
	// OS. Using filepath.Separator here would break discovery on Windows.
	const sep = "/"
	baseDir = strings.ReplaceAll(baseDir, `\`, sep)
	for _, configPath := range candidateConfigs {
		Debugf("[collectConfigsForStack %s] checking candidate config %q", callID, configPath)
		configDir := path.Dir(configPath)
		// needle is the directory prefix a frame must sit under for this config
		// to apply. It's matched at path-segment boundaries on both edges (see
		// below). configDir == "." means the config sits at the base root — a
		// parent/root config in the layering model — so it applies to every
		// frame under baseDir (and to every captured frame when baseDir is also
		// "."). Joining "." into the needle (e.g. "internal/./") would never
		// match a real frame path, silently dropping such configs.
		var needle string
		switch {
		case baseDir == "." && configDir == ".":
			needle = "" // root/parent config with baseDir ".": applies to any frame
		case baseDir == ".":
			needle = configDir + sep
		case configDir == ".":
			needle = baseDir + sep
		default:
			needle = baseDir + sep + configDir + sep
		}
		// Anchor the match on both edges. The trailing separator anchors the
		// right edge, so "service/amp" doesn't match "service/amplify/..." (and
		// acm/acmpca, account/accountaccess, bedrock/bedrockagent, ...). The
		// leading edge requires the needle at the start of the path or
		// immediately after a separator, so "notinternal/service/amp/" (or
		// "notservice/amp/" in baseDir "." mode) isn't mistaken for a frame under
		// the configured root. Frames always carry a file name, so the trailing
		// separator is present for a legitimate match. An empty needle (a root
		// config in baseDir "." mode) matches any captured frame.
		for _, stackPath := range relStackPaths {
			if needle == "" || strings.HasPrefix(stackPath, needle) || strings.Contains(stackPath, sep+needle) {
				cfg, err := loadConfigFile(ctx, fsys, configPath)
				if err != nil {
					Debugf("[collectConfigsForStack %s] error loading config %s: %v", callID, configPath, err)
					return nil, fmt.Errorf("error loading config %s: %w", configPath, err)
				}
				cfgsWithPaths = append(cfgsWithPaths, configWithPath{cfg, configPath, false})
				Debugf("[collectConfigsForStack %s] matched config %q for stack path %q", callID, configPath, stackPath)
				break // Only need to match once per config
			}
			Debugf("[collectConfigsForStack %s] config %q did not match, stackPath (%s) does not contain needle (%s)", callID, configPath, stackPath, needle)
		}
	}
	// Sort least specific first, most specific last, so mergeConfigs applies
	// global -> parent -> local (later entries override earlier). The designated
	// global config is "more global than a parent" per the layering model, so it
	// always sorts first even though a root candidate ("smarterr.hcl", depth 0)
	// is shallower than "smarterr/smarterr.hcl" (depth 1). SliceStable keeps a
	// deterministic order among equal-depth candidates.
	sort.SliceStable(cfgsWithPaths, func(i, j int) bool {
		a, b := cfgsWithPaths[i], cfgsWithPaths[j]
		if a.global != b.global {
			return a.global // global config sorts before everything else
		}
		return strings.Count(a.path, sep) < strings.Count(b.path, sep)
	})
	var configs []*Config
	for _, c := range cfgsWithPaths {
		configs = append(configs, c.cfg)
	}
	return configs, nil
}

// findAllConfigPaths scans the FS for all smarterr.hcl files, returning the global config path and other candidates.
func findAllConfigPaths(ctx context.Context, fsys FileSystem) (globalConfig string, candidateConfigs []string, err error) {
	callID := globalCallID(ctx)
	Debugf("[findAllConfigPaths %s] scanning filesystem for config files", callID)
	err = fsys.WalkDir(".", func(walkPath string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		// Match the exact file name, not a suffix: HasSuffix would also accept
		// "notsmarterr.hcl", and a root-level file like that would be applied to
		// every matching stack (see collectConfigsForStack's root-config
		// handling). ConfigFileName is documented as the file name.
		if path.Base(walkPath) != ConfigFileName {
			return nil
		}
		if strings.HasPrefix(walkPath, "smarterr/") {
			globalConfig = walkPath
		} else {
			candidateConfigs = append(candidateConfigs, walkPath)
		}
		return nil
	})
	Debugf("[findAllConfigPaths %s] found globalConfig=%q candidateConfigs=%v", callID, globalConfig, candidateConfigs)
	return
}

// loadConfigFile loads a single config file from the FS and parses it into a Config struct.
func loadConfigFile(ctx context.Context, fsys FileSystem, path string) (*Config, error) {
	callID := globalCallID(ctx)
	Debugf("[loadConfigFile %s] loading config file %q", callID, path)
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
	defer func() { _ = f.Close() }()
	stat, err := f.Stat()
	return err == nil && !stat.IsDir()
}
