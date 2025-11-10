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
				Name:        "DeprecatedEnrichAppend",
				Description: "smerr.EnrichAppend(...) -> smerr.AddEnrich(...)",
				Replace:     replaceDeprecatedEnrichAppend,
			},
			{
				Name:        "VariadicAppend",
				Description: "response.Diagnostics.Append(...) -> smerr.AddEnrich(...)",
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
				Name:        "AddErrorFmtSprintf",
				Description: "response.Diagnostics.AddError(..., fmt.Sprintf(...)) -> smerr.AddError(..., fmt.Errorf(...))",
				Replace:     replaceAddErrorFmtSprintf,
			},
			{
				Name:        "CreateProblemStandardMessage",
				Description: "response.Diagnostics.AddError with create.ProblemStandardMessage",
				Replace:     replaceCreateProblemStandardMessage,
			},
		},
	}
}

// replaceDeprecatedEnrichAppend handles deprecated smerr.EnrichAppend usage
func replaceDeprecatedEnrichAppend(content string) string {
	re := regexp.MustCompile(`(?m)(\s+)smerr\.EnrichAppend\(`)
	return re.ReplaceAllString(content, `${1}smerr.AddEnrich(`)
}

// replaceVariadicAppend handles response.Diagnostics.Append with ... variadic operator
func replaceVariadicAppend(content string) string {
	re := regexp.MustCompile(`(?m)(\s+)(resp|response)\.Diagnostics\.Append\((.+)\.\.\.\)$`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the parts
		submatches := re.FindStringSubmatch(match)
		if len(submatches) != 4 {
			return match
		}
		indent := submatches[1]
		respVar := submatches[2]
		arg := submatches[3]

		// Replace with smerr.AddEnrich (updated from deprecated EnrichAppend)
		return indent + "smerr.AddEnrich(ctx, &" + respVar + ".Diagnostics, " + arg + ")"
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
			return indent + "smerr.AddOne(ctx, &response.Diagnostics, " + fwdiagCall + ")"
		}

		return match // Return unchanged if we can't handle it
	})
}

// replaceCreateProblemStandardMessage handles create.ProblemStandardMessage patterns
func replaceCreateProblemStandardMessage(content string) string {
	// Handle cases with err.Error() - both simple and complex nested parentheses
	re1 := regexp.MustCompile(`(?s)(\s+)response\.Diagnostics\.AddError\(\s*create\.ProblemStandardMessage\([^)]*(?:\([^)]*\)[^)]*)*\),\s*([a-zA-Z_][a-zA-Z0-9_]*)\.Error\(\)\s*,?\s*\)`)
	content = re1.ReplaceAllString(content, `${1}smerr.AddError(ctx, &response.Diagnostics, $2)`)

	// Handle cases with errors.New("...").Error()
	re2 := regexp.MustCompile(`(?s)(\s+)response\.Diagnostics\.AddError\(\s*create\.ProblemStandardMessage\([^)]*(?:\([^)]*\)[^)]*)*\),\s*(errors\.New\([^)]*\))\.Error\(\)\s*,?\s*\)`)
	content = re2.ReplaceAllString(content, `${1}smerr.AddError(ctx, &response.Diagnostics, $2)`)

	return content
}

// replaceAddErrorFmtSprintf handles response.Diagnostics.AddError with fmt.Sprintf patterns
func replaceAddErrorFmtSprintf(content string) string {
	// Handle multiline response.Diagnostics.AddError calls
	re := regexp.MustCompile(`(?s)(\s+)(resp|response)\.Diagnostics\.AddError\(\s*"[^"]*",\s*(fmt\.Sprintf\([^)]*(?:\([^)]*\)[^)]*)*\)|"[^"]*")\s*,?\s*\)`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) != 4 {
			return match
		}
		indent := submatches[1]
		respVar := submatches[2]
		errorArg := submatches[3]

		// Convert fmt.Sprintf to fmt.Errorf, or wrap string literals in fmt.Errorf
		if strings.HasPrefix(errorArg, "fmt.Sprintf") {
			errorArg = strings.Replace(errorArg, "fmt.Sprintf", "fmt.Errorf", 1)
		} else if strings.HasPrefix(errorArg, `"`) {
			errorArg = "fmt.Errorf(" + errorArg + ")"
		}

		return indent + "smerr.AddError(ctx, &" + respVar + ".Diagnostics, " + errorArg + ")"
	})
}
