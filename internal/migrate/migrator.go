package migrate

import (
	"cmp"
	"regexp"
	"slices"
)

// Pattern represents a single transformation rule
type Pattern struct {
	Name        string
	Description string
	Regex       *regexp.Regexp
	Replace     func(string) string // For complex replacements
	Template    string              // For simple replacements
}

// PatternGroup represents a logical group of related patterns
type PatternGroup struct {
	Name     string
	Patterns []Pattern
	Order    int // Execution order
}

// MigratorOptions configures the migration behavior
type MigratorOptions struct {
	DryRun  bool
	Verbose bool
}

// Migrator handles the overall migration process
type Migrator struct {
	patterns []PatternGroup
	options  MigratorOptions
}

// NewMigrator creates a new migrator with the given options
func NewMigrator(opts MigratorOptions) *Migrator {
	return &Migrator{
		patterns: LoadPatterns(),
		options:  opts,
	}
}

// MigrateContent applies all pattern groups to the content in order
func (m *Migrator) MigrateContent(content string) string {
	// Sort pattern groups by execution order
	slices.SortFunc(m.patterns, func(a, b PatternGroup) int {
		return cmp.Compare(a.Order, b.Order)
	})

	// Apply pattern transformations
	for _, group := range m.patterns {
		content = m.applyPatternGroup(content, group)
	}

	// Add required imports after transformations
	importManager := NewImportManager(content)
	content = importManager.AddRequiredImports()

	return content
}

// applyPatternGroup applies all patterns in a group to the content
func (m *Migrator) applyPatternGroup(content string, group PatternGroup) string {
	for _, pattern := range group.Patterns {
		if pattern.Replace != nil {
			// Use custom replacement function
			content = pattern.Replace(content)
		} else if pattern.Regex != nil && pattern.Template != "" {
			// Use regex replacement with template
			content = pattern.Regex.ReplaceAllString(content, pattern.Template)
		}
	}
	return content
}
