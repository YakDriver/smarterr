package migrate

import (
	"regexp"
	"strings"
)

// CreateFrameworkPatterns creates patterns for Terraform Plugin Framework
func CreateFrameworkPatterns() PatternGroup {
	return PatternGroup{
		Name:  "FrameworkPatterns",
		Order: 3,
		Patterns: []Pattern{
			{
				Name:        "VariadicAppend",
				Description: "response.Diagnostics.Append(...) -> smerr.EnrichAppend(...)",
				Replace:     replaceVariadicAppend,
			},
			{
				Name:        "FwdiagAppend",
				Description: "response.Diagnostics.Append(fwdiag.*) -> smerr.* patterns",
				Replace:     replaceFwdiagAppend,
			},
			{
				Name:        "AddErrorSimple",
				Description: "response.Diagnostics.AddError(..., err.Error()) -> smerr.AddError(..., err)",
				Regex:       regexp.MustCompile(`(?m)(\s+)response\.Diagnostics\.AddError\(\s*"([^"]*)",\s*([^)]+)\.Error\(\)\s*\)$`),
				Template:    `${1}smerr.AddError(ctx, &response.Diagnostics, $3)`,
			},
			{
				Name:        "CreateProblemStandardMessage",
				Description: "response.Diagnostics.AddError with create.ProblemStandardMessage",
				Replace:     replaceCreateProblemStandardMessage,
			},
		},
	}
}

// replaceVariadicAppend handles response.Diagnostics.Append with ... variadic operator
func replaceVariadicAppend(content string) string {
	re := regexp.MustCompile(`(?m)(\s+)response\.Diagnostics\.Append\((.+)\.\.\.\)$`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the parts
		submatches := re.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}
		indent := submatches[1]
		arg := submatches[2]

		// Replace with smerr.EnrichAppend
		return indent + "smerr.EnrichAppend(ctx, &response.Diagnostics, " + arg + ")"
	})
}

// replaceFwdiagAppend handles response.Diagnostics.Append with fwdiag patterns
func replaceFwdiagAppend(content string) string {
	// Handle nested parentheses for fwdiag calls
	re := regexp.MustCompile(`(?m)(\s+)response\.Diagnostics\.Append\((fwdiag\.[^(]+\([^)]*\))\)$`)
	
	return re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}
		indent := submatches[1]
		fwdiagCall := submatches[2]
		
		// Check if it's a single diagnostic call
		if strings.Contains(fwdiagCall, "fwdiag.New") {
			return indent + "smerr.AppendOne(ctx, &response.Diagnostics, " + fwdiagCall + ")"
		}
		
		return match // Return unchanged if we can't handle it
	})
}

// replaceCreateProblemStandardMessage handles create.ProblemStandardMessage patterns
func replaceCreateProblemStandardMessage(content string) string {
	// Handle create.ProblemStandardMessage in AddError calls
	re := regexp.MustCompile(`(?m)(\s+)response\.Diagnostics\.AddError\(\s*create\.ProblemStandardMessage\([^)]*\),\s*([^)]+)\)$`)
	
	return re.ReplaceAllString(content, `${1}smerr.AddError(ctx, &response.Diagnostics, $2)`)
}
