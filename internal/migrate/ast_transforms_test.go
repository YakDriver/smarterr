package migrate

import (
	"strings"
	"testing"
)

func TestReplaceSDKResourceNotFoundAST(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "basic pattern with intretry.NotFound first",
			input: `package test

func test() {
	if intretry.NotFound(err) && !d.IsNewResource() {
		log.Printf("[WARN] Resource not found")
		d.SetId("")
		return diags
	}
}`,
			expected: `package test

func test() {
	if !d.IsNewResource() && intretry.NotFound(err) {
		smerr.AppendOne(ctx, diags, sdkdiag.NewResourceNotFoundWarningDiagnostic(err))
		d.SetId("")
		return diags
	}
}`,
		},
		{
			name: "basic pattern with !d.IsNewResource first",
			input: `package test

func test() {
	if !d.IsNewResource() && intretry.NotFound(err) {
		log.Printf("[WARN] Resource not found")
		d.SetId("")
		return diags
	}
}`,
			expected: `package test

func test() {
	if !d.IsNewResource() && intretry.NotFound(err) {
		smerr.AppendOne(ctx, diags, sdkdiag.NewResourceNotFoundWarningDiagnostic(err))
		d.SetId("")
		return diags
	}
}`,
		},
		{
			name: "complex log.Printf with format args",
			input: `package test

func test() {
	if intretry.NotFound(err) && !d.IsNewResource() {
		log.Printf("[WARN] AppSync Datasource %q not found, removing from state", d.Id())
		d.SetId("")
		return diags
	}
}`,
			expected: `package test

func test() {
	if !d.IsNewResource() && intretry.NotFound(err) {
		smerr.AppendOne(ctx, diags, sdkdiag.NewResourceNotFoundWarningDiagnostic(err))
		d.SetId("")
		return diags
	}
}`,
		},
		{
			name: "no transformation - wrong pattern",
			input: `package test

func test() {
	if someOtherCondition {
		log.Printf("[WARN] Something else")
		d.SetId("")
		return diags
	}
}`,
			expected: `package test

func test() {
	if someOtherCondition {
		log.Printf("[WARN] Something else")
		d.SetId("")
		return diags
	}
}`,
		},
		{
			name: "no transformation - wrong number of statements",
			input: `package test

func test() {
	if intretry.NotFound(err) && !d.IsNewResource() {
		log.Printf("[WARN] Resource not found")
		return diags
	}
}`,
			expected: `package test

func test() {
	if intretry.NotFound(err) && !d.IsNewResource() {
		log.Printf("[WARN] Resource not found")
		return diags
	}
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceSDKResourceNotFoundAST(tt.input)
			
			// Normalize whitespace for comparison
			normalizeWhitespace := func(s string) string {
				lines := strings.Split(s, "\n")
				var normalized []string
				for _, line := range lines {
					if strings.TrimSpace(line) != "" {
						normalized = append(normalized, strings.TrimSpace(line))
					}
				}
				return strings.Join(normalized, "\n")
			}
			
			if normalizeWhitespace(result) != normalizeWhitespace(tt.expected) {
				t.Errorf("replaceSDKResourceNotFoundAST() =\n%s\n\nwant:\n%s", result, tt.expected)
			}
		})
	}
}

func TestSDKResourceNotFoundTransformer_isSDKResourceNotFoundPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name: "valid pattern - intretry.NotFound first",
			input: `if intretry.NotFound(err) && !d.IsNewResource() {
				log.Printf("[WARN] test")
				d.SetId("")
				return diags
			}`,
			expected: true,
		},
		{
			name: "valid pattern - !d.IsNewResource first",
			input: `if !d.IsNewResource() && intretry.NotFound(err) {
				log.Printf("[WARN] test")
				d.SetId("")
				return diags
			}`,
			expected: true,
		},
		{
			name: "invalid - wrong condition",
			input: `if someOther && !d.IsNewResource() {
				log.Printf("[WARN] test")
				d.SetId("")
				return diags
			}`,
			expected: false,
		},
		{
			name: "invalid - wrong number of statements",
			input: `if intretry.NotFound(err) && !d.IsNewResource() {
				log.Printf("[WARN] test")
				return diags
			}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test would require parsing the input as an if statement
			// For now, we test the full transformation which exercises the pattern matching
			fullInput := `package test
func test() {
	` + tt.input + `
}`
			result := replaceSDKResourceNotFoundAST(fullInput)
			
			// If pattern should match, result should be different from input
			// If pattern shouldn't match, result should be same as input
			changed := result != fullInput
			if changed != tt.expected {
				t.Errorf("Pattern matching for %q: got changed=%v, want changed=%v", tt.name, changed, tt.expected)
			}
		})
	}
}
