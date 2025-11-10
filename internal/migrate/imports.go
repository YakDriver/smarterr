package migrate

import (
	"go/parser"
	"go/token"
	"maps"
	"regexp"
	"strings"
)

// ImportManager handles Go import management for migrations
type ImportManager struct {
	content string
}

// NewImportManager creates a new import manager for the given content
func NewImportManager(content string) *ImportManager {
	return &ImportManager{content: content}
}

// RequiredImports defines the imports needed for smarterr migrations
var RequiredImports = []ImportSpec{
	{
		Path: "github.com/YakDriver/smarterr",
		Name: "",
	},
	{
		Path: "github.com/hashicorp/terraform-provider-aws/internal/smerr",
		Name: "",
	},
	{
		Path: "github.com/hashicorp/terraform-provider-aws/internal/errs/sdkdiag",
		Name: "",
	},
}

// ImportSpec represents an import specification
type ImportSpec struct {
	Path string // Import path
	Name string // Import name/alias (empty for default)
}

// ConflictingImports defines import conflicts that need special handling
var ConflictingImports = map[string]ConflictResolution{
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry": {
		ConflictsWith: "github.com/hashicorp/terraform-provider-aws/internal/retry",
		Resolution: ImportSpec{
			Path: "github.com/hashicorp/terraform-provider-aws/internal/retry",
			Name: "intretry",
		},
		PrefixMapping: map[string]string{
			"retry": "intretry",
		},
	},
}

// ConflictResolution defines how to resolve import conflicts
type ConflictResolution struct {
	ConflictsWith string            // The import path that conflicts
	Resolution    ImportSpec        // How to resolve the conflict
	PrefixMapping map[string]string // How to change prefixes in code
}

// AddRequiredImports adds all required imports for smarterr migrations
func (im *ImportManager) AddRequiredImports() string {
	content := im.content

	for _, importSpec := range RequiredImports {
		if !im.hasImport(importSpec.Path) {
			content = im.addImport(content, importSpec)
		}
	}

	return content
}

// ResolveImportConflicts resolves known import conflicts and returns the prefix mapping
func (im *ImportManager) ResolveImportConflicts() (string, map[string]string) {
	content := im.content
	prefixMappings := make(map[string]string)

	for conflictingPath, resolution := range ConflictingImports {
		if im.hasImport(conflictingPath) && !im.hasImport(resolution.ConflictsWith) {
			// Add the aliased import to resolve the conflict
			content = im.addImport(content, resolution.Resolution)

			// Merge prefix mappings
			maps.Copy(prefixMappings, resolution.PrefixMapping)
		}
	}

	return content, prefixMappings
}

// GetRetryPrefix returns the appropriate retry prefix based on import conflicts
func (im *ImportManager) GetRetryPrefix() string {
	wrongRetry := "github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	correctRetry := "github.com/hashicorp/terraform-provider-aws/internal/retry"

	if im.hasImport(wrongRetry) && !im.hasImport(correctRetry) {
		return "intretry" // Use alias when there's a conflict
	}
	return "retry" // Default prefix
}

// AddImportWithAlias adds an import with a specific alias
func (im *ImportManager) AddImportWithAlias(path, alias string) string {
	if im.hasImport(path) {
		return im.content
	}

	return im.addImport(im.content, ImportSpec{Path: path, Name: alias})
}

// hasImport checks if an import path already exists in the content
func (im *ImportManager) hasImport(path string) bool {
	// Use AST parsing for more accurate detection
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", im.content, parser.ImportsOnly)
	if err != nil {
		// Fallback to string matching if AST parsing fails
		return strings.Contains(im.content, `"`+path+`"`)
	}

	for _, imp := range file.Imports {
		if imp.Path.Value == `"`+path+`"` {
			return true
		}
	}

	return false
}

// hasImportWithAlias checks if an import path already exists with a specific alias
func (im *ImportManager) hasImportWithAlias(path, alias string) bool {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", im.content, parser.ImportsOnly)
	if err != nil {
		// Fallback to string matching
		return strings.Contains(im.content, alias+` "`+path+`"`)
	}

	for _, imp := range file.Imports {
		if imp.Path.Value == `"`+path+`"` && imp.Name != nil && imp.Name.Name == alias {
			return true
		}
	}
	return false
}

// removeImport removes an import from the content
func (im *ImportManager) removeImport(content, path string) string {
	// Simple regex-based removal for now
	patterns := []string{
		`\s*"` + regexp.QuoteMeta(path) + `"\s*\n`,
		`\s*"` + regexp.QuoteMeta(path) + `"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		content = re.ReplaceAllString(content, "")
	}

	return content
}

// addImport adds a single import to the content using the same logic as the original
func (im *ImportManager) addImport(content string, spec ImportSpec) string {
	// Check if import already exists to prevent duplicates
	tempIM := NewImportManager(content)
	if tempIM.hasImport(spec.Path) || (spec.Name != "" && tempIM.hasImportWithAlias(spec.Path, spec.Name)) {
		return content
	}

	// Use the same regex pattern as the original addInternalRetryImport for consistency
	importBlockPattern := regexp.MustCompile(`(import \(\n)((?:[^\)]*\n)*?)(\))`)
	matches := importBlockPattern.FindStringSubmatch(content)
	if len(matches) != 4 {
		return content // No import block found
	}

	imports := matches[2]

	// Construct import line with proper indentation (matching original behavior)
	var importLine string
	if spec.Name != "" {
		importLine = "\t" + spec.Name + ` "` + spec.Path + `"` + "\n"
	} else {
		importLine = "\t" + `"` + spec.Path + `"` + "\n"
	}

	imports += importLine

	// Reconstruct the import block (matching original behavior)
	return strings.Replace(content, matches[0], matches[1]+imports+matches[3], 1)
}

// GetImports returns all imports found in the content using AST parsing
func (im *ImportManager) GetImports() ([]ImportInfo, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", im.content, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	var imports []ImportInfo
	for _, imp := range file.Imports {
		info := ImportInfo{
			Path: strings.Trim(imp.Path.Value, `"`),
		}
		if imp.Name != nil {
			info.Name = imp.Name.Name
		}
		imports = append(imports, info)
	}

	return imports, nil
}

// ImportInfo represents information about an import
type ImportInfo struct {
	Path string // Import path without quotes
	Name string // Import name/alias (empty for default)
}

// HasConflictingRetryImport checks if there's a conflicting retry import (legacy method for compatibility)
func (im *ImportManager) HasConflictingRetryImport() bool {
	wrongRetry := "github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	correctRetry := "github.com/hashicorp/terraform-provider-aws/internal/retry"

	return im.hasImport(wrongRetry) && !im.hasImport(correctRetry)
}

// AddAliasedRetryImport adds the internal retry import with an alias (legacy method for compatibility)
func (im *ImportManager) AddAliasedRetryImport() string {
	return im.AddImportWithAlias(
		"github.com/hashicorp/terraform-provider-aws/internal/retry",
		"intretry",
	)
}

// CreateImportPatterns creates patterns for import-related transformations
func CreateImportPatterns() PatternGroup {
	return PatternGroup{
		Name:  "ImportPatterns",
		Order: 0, // Run first, before other patterns
		Patterns: []Pattern{
			{
				Name:        "AddRequiredImports",
				Description: "Add required smarterr and smerr imports",
				Replace:     addRequiredImports,
			},
		},
	}
}

// addRequiredImports is the pattern function that adds required imports
func addRequiredImports(content string) string {
	im := NewImportManager(content)
	return im.AddRequiredImports()
}
