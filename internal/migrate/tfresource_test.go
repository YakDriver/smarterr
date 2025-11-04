package migrate

import "testing"

func TestCreateTfresourcePatterns(t *testing.T) {
	patterns := CreateTfresourcePatterns()
	
	if patterns.Name != "TfresourcePatterns" {
		t.Errorf("Expected name 'TfresourcePatterns', got %s", patterns.Name)
	}
	
	if patterns.Order != 2 {
		t.Errorf("Expected order 2, got %d", patterns.Order)
	}
	
	if len(patterns.Patterns) == 0 {
		t.Error("Expected patterns to be non-empty")
	}
}

func TestReplaceTfresourceNotFound(t *testing.T) {
	migrator := NewMigrator(MigratorOptions{})
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "standalone fwdiag.NewResourceNotFoundWarningDiagnostic",
			input: `	response.Diagnostics.Append(fwdiag.NewResourceNotFoundWarningDiagnostic(err))`,
			expected: `	smerr.AppendOne(ctx, &response.Diagnostics, fwdiag.NewResourceNotFoundWarningDiagnostic(err))`,
		},
		{
			name: "tfresource.NotFound with diagnostic",
			input: `	if tfresource.NotFound(err) {
		response.Diagnostics.Append(fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		response.State.RemoveResource(ctx)
		return
	}`,
			expected: `	if retry.NotFound(err) {
		smerr.AppendOne(ctx, &response.Diagnostics, fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		response.State.RemoveResource(ctx)
		return
	}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrator.MigrateContent(tt.input)
			if result != tt.expected {
				t.Errorf("MigrateContent() =\n%q\nwant:\n%q", result, tt.expected)
			}
		})
	}
}
