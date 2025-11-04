package migrate

import "regexp"

// CreateSDKv2Patterns creates patterns for Terraform Plugin SDKv2
func CreateSDKv2Patterns() PatternGroup {
	return PatternGroup{
		Name:  "SDKv2Patterns",
		Order: 4,
		Patterns: []Pattern{
			{
				Name:        "AppendFromErr",
				Description: "sdkdiag.AppendFromErr -> smerr.Append",
				Regex:       regexp.MustCompile(`sdkdiag\.AppendFromErr\(([^,]+),\s*([^)]+)\)`),
				Template:    `smerr.Append(ctx, $1, $2)`,
			},
			{
				Name:        "AppendErrorfWithID",
				Description: "sdkdiag.AppendErrorf with ID -> smerr.Append with smerr.ID",
				Regex:       regexp.MustCompile(`(?m)sdkdiag\.AppendErrorf\(([^,]+),\s*"[^"]*",\s*([^,\n]+),\s*([^,\n]+)\)$`),
				Template:    `smerr.Append(ctx, $1, $3, smerr.ID, $2)`,
			},
			{
				Name:        "AppendErrorfSimple",
				Description: "sdkdiag.AppendErrorf simple -> smerr.Append",
				Regex:       regexp.MustCompile(`(?m)sdkdiag\.AppendErrorf\(([^,]+),\s*"[^"]*",\s*([^,\n]+)\)$`),
				Template:    `smerr.Append(ctx, $1, $2)`,
			},
			{
				Name:        "CreateAppendDiagError",
				Description: "create.AppendDiagError -> smerr.Append",
				Regex:       regexp.MustCompile(`(?m)create\.AppendDiagError\(([^,]+),\s*[^)]*\)$`),
				Template:    `smerr.Append(ctx, $1, err, smerr.ID, id)`,
			},
			{
				Name:        "CreateAddError",
				Description: "create.AddError -> smerr.AddError",
				Regex:       regexp.MustCompile(`(?m)create\.AddError\(&([^,]+),\s*[^)]*\)$`),
				Template:    `smerr.AddError(ctx, &$1, err, smerr.ID, id)`,
			},
		},
	}
}
