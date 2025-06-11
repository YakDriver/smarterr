// merge.go
// Config merging logic for smarterr
package internal

// mergeConfigs merges a slice of Configs, from least to most specific.
func mergeConfigs(configs []*Config) *Config {
	if len(configs) == 0 {
		return &Config{}
	}
	merged := configs[0]
	for i := 1; i < len(configs); i++ {
		mergeConfigsPair(merged, configs[i])
	}
	return merged
}

// mergeConfigsPair merges two Config objects: add takes precedence over base.
//
// 1. SmarterrDebug and TokenErrorMode are overwritten by add if set.
// 2. Tokens are merged by name.
// 3. Hints, Parameters, StackMatches, Templates, and Transforms are merged by name.
func mergeConfigsPair(base *Config, add *Config) {
	// Overwrite SmarterrDebug and TokenErrorMode if set in add
	if add.SmarterrDebug != nil {
		base.SmarterrDebug = add.SmarterrDebug
	}
	if add.TokenErrorMode != "" {
		base.TokenErrorMode = add.TokenErrorMode
	}

	// Merge tokens: replace by name only (positions removed)
	nameMap := map[string]int{} // Map name to index in base.Tokens

	// Build map for existing tokens in base
	for i, t := range base.Tokens {
		nameMap[t.Name] = i
	}

	// Add or replace tokens from add
	for _, t := range add.Tokens {
		if i, ok := nameMap[t.Name]; ok {
			// Overwrite token with the same name
			base.Tokens[i] = t
		} else {
			// Add new token
			base.Tokens = append(base.Tokens, t)
		}
		// Update map to reflect the new token
		nameMap[t.Name] = len(base.Tokens) - 1
	}

	// Merge hints: replace by name
	hintMap := map[string]int{} // Map name to index in base.Hints

	// Build map for existing hints in base
	for i, h := range base.Hints {
		hintMap[h.Name] = i
	}

	// Add or replace hints from add
	for _, h := range add.Hints {
		if i, ok := hintMap[h.Name]; ok {
			// Overwrite hint with the same name
			base.Hints[i] = h
		} else {
			// Add new hint
			base.Hints = append(base.Hints, h)
		}

		// Update map to reflect the new hint
		hintMap[h.Name] = len(base.Hints) - 1
	}

	// Merge parameters: replace by name
	parameterMap := map[string]int{} // Map name to index in base.Parameters

	// Build map for existing parameters in base
	for i, p := range base.Parameters {
		parameterMap[p.Name] = i
	}

	// Add or replace parameters from add
	for _, p := range add.Parameters {
		if i, ok := parameterMap[p.Name]; ok {
			// Overwrite parameter with the same name
			base.Parameters[i] = p
		} else {
			// Add new parameter
			base.Parameters = append(base.Parameters, p)
		}

		// Update map to reflect the new parameter
		parameterMap[p.Name] = len(base.Parameters) - 1
	}

	// Merge stack matches: replace by name
	stackMatchMap := map[string]int{} // Map name to index in base.StackMatches

	// Build map for existing stack matches in base
	for i, sm := range base.StackMatches {
		stackMatchMap[sm.Name] = i
	}

	// Add or replace stack matches from add
	for _, sm := range add.StackMatches {
		if i, ok := stackMatchMap[sm.Name]; ok {
			// Overwrite stack match with the same name
			base.StackMatches[i] = sm
		} else {
			// Add new stack match
			base.StackMatches = append(base.StackMatches, sm)
		}

		// Update map to reflect the new stack match
		stackMatchMap[sm.Name] = len(base.StackMatches) - 1
	}

	// Merge templates: replace by name
	tmplMap := map[string]int{} // Map name to index in base.Templates
	for i, tmpl := range base.Templates {
		tmplMap[tmpl.Name] = i
	}
	for _, tmpl := range add.Templates {
		if i, ok := tmplMap[tmpl.Name]; ok {
			base.Templates[i] = tmpl
		} else {
			base.Templates = append(base.Templates, tmpl)
		}
		tmplMap[tmpl.Name] = len(base.Templates) - 1
	}

	// Merge transforms: replace by name
	trMap := map[string]int{} // Map name to index in base.Transforms
	for i, tr := range base.Transforms {
		trMap[tr.Name] = i
	}
	for _, tr := range add.Transforms {
		if i, ok := trMap[tr.Name]; ok {
			base.Transforms[i] = tr
		} else {
			base.Transforms = append(base.Transforms, tr)
		}
		trMap[tr.Name] = len(base.Transforms) - 1
	}
}
