package migrate

import (
	"regexp"
	"strings"
)

// CreateBareErrorPatterns creates patterns for bare error returns
func CreateBareErrorPatterns() PatternGroup {
	return PatternGroup{
		Name:  "BareErrorReturns",
		Order: 1,
		Patterns: []Pattern{
			{
				Name:        "DeprecatedSmarterrEnrichAppend",
				Description: "smarterr.EnrichAppend(...) -> smarterr.AddEnrich(...)",
				Regex:       regexp.MustCompile(`(?m)(\s+)smarterr\.EnrichAppend\(`),
				Template:    `${1}smarterr.AddEnrich(`,
			},
			{
				Name:        "SimpleReturn",
				Description: "return nil, err -> return nil, smarterr.NewError(err)",
				Regex:       regexp.MustCompile(`(?m)(\s+)return nil, err$`),
				Template:    `${1}return nil, smarterr.NewError(err)`,
			},
			{
				Name:        "NonNilReturn",
				Description: "return <value>, err -> return <value>, smarterr.NewError(err)",
				Regex:       regexp.MustCompile(`(?m)(\s+)return ([^,\n]+), err$`),
				Template:    `${1}return $2, smarterr.NewError(err)`,
			},
			{
				Name:        "TfresourceNewEmptyResultError",
				Description: "return nil, tfresource.NewEmptyResultError(...) -> smarterr.NewError(...)",
				Regex:       regexp.MustCompile(`(?m)(\s+)return nil, tfresource\.NewEmptyResultError\(([^)]*)\)$`),
				Template:    `${1}return nil, smarterr.NewError(tfresource.NewEmptyResultError($2))`,
			},
			{
				Name:        "TfresourceAssertSingleValueResult",
				Description: "return tfresource.AssertSingleValueResult(...) -> smarterr.Assert(...)",
				Regex:       regexp.MustCompile(`(\s+)return tfresource\.AssertSingleValueResult\(([^)]+)\)`),
				Template:    `${1}return smarterr.Assert(tfresource.AssertSingleValueResult($2))`,
			},
			{
				Name:        "RetryNotFoundErrorMultiLine",
				Description: "Multi-line retry.NotFoundError -> smarterr.NewError(...)",
				Regex:       regexp.MustCompile(`(?m)(\s+)return nil, &retry\.NotFoundError\{\s*\n\s*LastError:\s*([^,\n]+),\s*\n\s*LastRequest:\s*([^,\n]+),?\s*\n\s*\}$`),
				Template:    `${1}return nil, smarterr.NewError(&retry.NotFoundError{LastError: $2, LastRequest: $3})`,
			},
			{
				Name:        "RetryNotFoundErrorSingleLine",
				Description: "Single-line retry.NotFoundError -> smarterr.NewError(...)",
				Regex:       regexp.MustCompile(`(?m)(\s+)return nil, &retry\.NotFoundError\{\s*LastError:\s*([^,}]+),\s*LastRequest:\s*([^,}]+),?\s*\}$`),
				Template:    `${1}return nil, smarterr.NewError(&retry.NotFoundError{LastError: $2, LastRequest: $3})`,
			},
			{
				Name:        "FmtErrorfNewError",
				Description: "return ..., fmt.Errorf(...) -> return ..., smarterr.NewError(fmt.Errorf(...))",
				Regex:       regexp.MustCompile(`(\s+)return (.+), (fmt\.Errorf\([^)]+\))`),
				Template:    `${1}return $2, smarterr.NewError($3)`,
			},
			{
				Name:        "FmtErrorf",
				Description: "return fmt.Errorf(..., err) -> return smarterr.NewError(err)",
				Regex:       regexp.MustCompile(`(\s+)return fmt\.Errorf\(.*,\s*([^)]+)\)`),
				Template:    `${1}return smarterr.NewError($2)`,
			},
			{
				Name:        "StateRefreshFunc",
				Description: "return nil, \"\", err -> return nil, \"\", smarterr.NewError(err)",
				Regex:       regexp.MustCompile(`(?m)(\s+)return nil, "", err$`),
				Template:    `${1}return nil, "", smarterr.NewError(err)`,
			},
			{
				Name:        "UnexpectedFormatError",
				Description: "Wrap fmt.Errorf with unexpected format with smarterr.NewError",
				Regex:       regexp.MustCompile(`(?mi)(\s+return .+, )(fmt\.Errorf\("[^"]*unexpected format[^"]*".*?\))$`),
				Template:    `${1}smarterr.NewError($2)`,
			},
			{
				Name:        "DiagsAddError",
				Description: "Convert diags.AddError patterns to smarterr.NewError returns",
				Replace:     replaceDiagsAddError,
			},
		},
	}
}

// replaceDiagsAddError converts helper functions that use diags.AddError to return smarterr.NewError
func replaceDiagsAddError(content string) string {
	// Add missing var diags diag.Diagnostics declarations for functions that use diags but don't declare it
	funcPattern := regexp.MustCompile(`(?s)(func\s+[^{]*\([^)]*\)\s*\([^,)]+,\s*diag\.Diagnostics\s*\)\s*\{\s*)(\s*switch|\s*case|\s*[a-zA-Z])`)
	content = funcPattern.ReplaceAllStringFunc(content, func(match string) string {
		// Check if diags is used but not declared
		if strings.Contains(match, "diags") && !strings.Contains(match, "var diags") {
			submatches := funcPattern.FindStringSubmatch(match)
			if len(submatches) >= 3 {
				return submatches[1] + "var diags diag.Diagnostics\n\t" + submatches[2]
			}
		}
		return match
	})
	
	return content
}
