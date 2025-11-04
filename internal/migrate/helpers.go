package migrate

import "regexp"

// CreateHelperPatterns creates patterns for helper functions and standard library
func CreateHelperPatterns() PatternGroup {
	return PatternGroup{
		Name:  "HelperPatterns",
		Order: 5,
		Patterns: []Pattern{
			{
				Name:        "StandardLibraryAppend",
				Description: "append(diags, ...) -> smerr.AppendEnrich",
				Regex:       regexp.MustCompile(`(?m)(\s+)return append\(diags, (.+)\.\.\.\)$`),
				Template:    `${1}return smerr.AppendEnrich(ctx, diags, $2)`,
			},
		},
	}
}
