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
// - Smarterr (debug, token_error_mode) is overwritten by add if set.
// - Tokens, Hints, Parameters, StackMatches, Templates, and Transforms are merged by name (add replaces base).
func mergeConfigsPair(base *Config, add *Config) {
	// Overwrite Smarterr fields if set in add
	if add.Smarterr != nil {
		if base.Smarterr == nil {
			base.Smarterr = &Smarterr{}
		}
		if add.Smarterr.Debug {
			base.Smarterr.Debug = true
		}
		if add.Smarterr.TokenErrorMode != "" {
			base.Smarterr.TokenErrorMode = add.Smarterr.TokenErrorMode
		}
	}

	// Merge tokens by name (add replaces base)
	tokenMap := make(map[string]int)
	for i, t := range base.Tokens {
		tokenMap[t.Name] = i
	}
	for _, t := range add.Tokens {
		if i, ok := tokenMap[t.Name]; ok {
			base.Tokens[i] = t
		} else {
			base.Tokens = append(base.Tokens, t)
		}
		tokenMap[t.Name] = len(base.Tokens) - 1
	}

	// Merge hints by name (add replaces base)
	hintMap := make(map[string]int)
	for i, h := range base.Hints {
		hintMap[h.Name] = i
	}
	for _, h := range add.Hints {
		if i, ok := hintMap[h.Name]; ok {
			base.Hints[i] = h
		} else {
			base.Hints = append(base.Hints, h)
		}
		hintMap[h.Name] = len(base.Hints) - 1
	}

	// Merge parameters by name (add replaces base)
	paramMap := make(map[string]int)
	for i, p := range base.Parameters {
		paramMap[p.Name] = i
	}
	for _, p := range add.Parameters {
		if i, ok := paramMap[p.Name]; ok {
			base.Parameters[i] = p
		} else {
			base.Parameters = append(base.Parameters, p)
		}
		paramMap[p.Name] = len(base.Parameters) - 1
	}

	// Merge stack matches by name (add replaces base)
	stackMatchMap := make(map[string]int)
	for i, sm := range base.StackMatches {
		stackMatchMap[sm.Name] = i
	}
	for _, sm := range add.StackMatches {
		if i, ok := stackMatchMap[sm.Name]; ok {
			base.StackMatches[i] = sm
		} else {
			base.StackMatches = append(base.StackMatches, sm)
		}
		stackMatchMap[sm.Name] = len(base.StackMatches) - 1
	}

	// Merge templates by name (add replaces base)
	tmplMap := make(map[string]int)
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

	// Merge transforms by name (add replaces base)
	trMap := make(map[string]int)
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
