package migrate

import "testing"

func TestCreateBareErrorPatterns(t *testing.T) {
	patterns := CreateBareErrorPatterns()

	if patterns.Name != "BareErrorReturns" {
		t.Errorf("Expected name 'BareErrorReturns', got %s", patterns.Name)
	}

	if patterns.Order != 1 {
		t.Errorf("Expected order 1, got %d", patterns.Order)
	}

	if len(patterns.Patterns) == 0 {
		t.Error("Expected patterns to be non-empty")
	}
}

func TestDeprecatedSmarterrEnrichAppend(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "smarterr.EnrichAppend to smarterr.AddEnrich",
			input:    "\tsmarterr.EnrichAppend(ctx, diags, someFunc())",
			expected: "\tsmarterr.AddEnrich(ctx, diags, someFunc())",
		},
		{
			name:     "multiple smarterr.EnrichAppend calls",
			input:    "\tsmarterr.EnrichAppend(ctx, diags, func1())\n\tsmarterr.EnrichAppend(ctx, diags, func2())",
			expected: "\tsmarterr.AddEnrich(ctx, diags, func1())\n\tsmarterr.AddEnrich(ctx, diags, func2())",
		},
	}

	migrator := NewMigrator(MigratorOptions{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrator.MigrateContent(tt.input)
			if result != tt.expected {
				t.Errorf("MigrateContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestReplaceAssertSingleValueResult(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple case",
			input:    "\treturn tfresource.AssertSingleValueResult(result)",
			expected: "\treturn smarterr.Assert(tfresource.AssertSingleValueResult(result))",
		},
	}

	migrator := NewMigrator(MigratorOptions{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrator.MigrateContent(tt.input)
			if result != tt.expected {
				t.Errorf("MigrateContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNonNilReturn(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "return value, err",
			input:    "\treturn output, err",
			expected: "\treturn output, smarterr.NewError(err)",
		},
		{
			name:     "return complex value, err",
			input:    "\treturn result.Value, err",
			expected: "\treturn result.Value, smarterr.NewError(err)",
		},
	}

	migrator := NewMigrator(MigratorOptions{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrator.MigrateContent(tt.input)
			if result != tt.expected {
				t.Errorf("MigrateContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}
