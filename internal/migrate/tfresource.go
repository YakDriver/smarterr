package migrate

import (
	"regexp"
	"strings"
)

// CreateTfresourcePatterns creates patterns for tfresource-related transformations
func CreateTfresourcePatterns() PatternGroup {
	return PatternGroup{
		Name:  "TfresourcePatterns",
		Order: 2,
		Patterns: []Pattern{
			{
				Name:        "NotFoundAntiPatterns",
				Description: "tfresource.NotFound anti-patterns with proper import aliasing",
				Replace:     replaceTfresourceNotFound,
			},
			{
				Name:        "TfresourceNotFoundToIntretry",
				Description: "tfresource.NotFound -> intretry.NotFound",
				Regex:       regexp.MustCompile(`tfresource\.NotFound\(([^)]+)\)`),
				Template:    `intretry.NotFound($1)`,
			},
		},
	}
}

// replaceTfresourceNotFound handles tfresource.NotFound anti-patterns and import aliasing
func replaceTfresourceNotFound(content string) string {
	im := NewImportManager(content)
	
	// Use the specialized retry prefix logic (preserves exact original behavior)
	retryPrefix := im.GetRetryPrefix()
	if im.HasConflictingRetryImport() {
		// Need to add aliased import and use intretry prefix
		content = im.AddAliasedRetryImport()
		// Update the import manager with new content
		im = NewImportManager(content)
	}

	// Pattern 1: with diagnostic
	pattern1 := regexp.MustCompile(`(?s)(\n?\s*)if tfresource\.NotFound\(err\) \{\s*response\.Diagnostics\.Append\(fwdiag\.NewResourceNotFoundWarningDiagnostic\(err\)\)\s*response\.State\.RemoveResource\(ctx\)\s*return\s*\}`)
	content = pattern1.ReplaceAllStringFunc(content, func(match string) string {
		submatch := pattern1.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		indent := submatch[1]
		// Extract base indentation (remove all leading newlines)
		baseIndent := strings.TrimLeft(indent, "\n")
		// Preserve the original leading newlines
		leadingNewlines := strings.TrimSuffix(indent, baseIndent)
		return leadingNewlines + baseIndent + `if ` + retryPrefix + `.NotFound(err) {` + "\n" + baseIndent + "\t" + `smerr.AddOne(ctx, &response.Diagnostics, fwdiag.NewResourceNotFoundWarningDiagnostic(err))` + "\n" + baseIndent + "\t" + `response.State.RemoveResource(ctx)` + "\n" + baseIndent + "\t" + `return` + "\n" + baseIndent + `}`
	})

	// Pattern 2: without diagnostic
	pattern2 := regexp.MustCompile(`(?s)(\n?\s*)if tfresource\.NotFound\(err\) \{\s*response\.State\.RemoveResource\(ctx\)\s*return\s*\}`)
	content = pattern2.ReplaceAllStringFunc(content, func(match string) string {
		submatch := pattern2.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		indent := submatch[1]
		// Extract base indentation (remove all leading newlines)
		baseIndent := strings.TrimLeft(indent, "\n")
		// Preserve the original leading newlines
		leadingNewlines := strings.TrimSuffix(indent, baseIndent)
		return leadingNewlines + baseIndent + `if ` + retryPrefix + `.NotFound(err) {` + "\n" + baseIndent + "\t" + `smerr.AddOne(ctx, &response.Diagnostics, fwdiag.NewResourceNotFoundWarningDiagnostic(err))` + "\n" + baseIndent + "\t" + `response.State.RemoveResource(ctx)` + "\n" + baseIndent + "\t" + `return` + "\n" + baseIndent + `}`
	})

	// Handle standalone fwdiag.NewResourceNotFoundWarningDiagnostic calls
	content = regexp.MustCompile(`(?m)(\s+)response\.Diagnostics\.Append\(fwdiag\.NewResourceNotFoundWarningDiagnostic\(([^)]+)\)\)$`).
		ReplaceAllString(content, `${1}smerr.AddOne(ctx, &response.Diagnostics, fwdiag.NewResourceNotFoundWarningDiagnostic($2))`)

	return content
}
