package migrate

import "testing"

func TestMigrationDetector_NeedsMigration(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "no migration needed",
			content:  `package main\n\nfunc foo() {}`,
			expected: false,
		},
		{
			name:     "sdkdiag.AppendFromErr present",
			content:  `sdkdiag.AppendFromErr(diags, err)`,
			expected: true,
		},
		{
			name:     "sdkdiag.AppendErrorf present",
			content:  `sdkdiag.AppendErrorf(diags, "error", err)`,
			expected: true,
		},
		{
			name:     "response.Diagnostics.Append present",
			content:  `response.Diagnostics.Append(someFunc())`,
			expected: true,
		},
		{
			name:     "response.Diagnostics.AddError present",
			content:  `response.Diagnostics.AddError("msg", err.Error())`,
			expected: true,
		},
		{
			name:     "create.AppendDiagError present",
			content:  `create.AppendDiagError(diags, names.EC2, create.ErrActionCreating, ResNameVPC, id, err)`,
			expected: true,
		},
		{
			name:     "create.AddError present",
			content:  `create.AddError(&response.Diagnostics, names.EC2, create.ErrActionCreating, ResNameVPC, id, err)`,
			expected: true,
		},
		{
			name:     "bare error return present",
			content:  "func foo() error {\n\treturn nil, err\n}",
			expected: true,
		},
		{
			name:     "retry.NotFoundError present",
			content:  `return nil, &retry.NotFoundError{LastError: err}`,
			expected: true,
		},
		{
			name:     "tfresource.NewEmptyResultError present",
			content:  `return nil, tfresource.NewEmptyResultError(input)`,
			expected: true,
		},
		{
			name:     "tfresource.AssertSingleValueResult present",
			content:  `return tfresource.AssertSingleValueResult(output.Subnets)`,
			expected: true,
		},
	}

	detector := NewMigrationDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.NeedsMigration(tt.content)
			if result != tt.expected {
				t.Errorf("NeedsMigration() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNeedsMigration(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "convenience function - needs migration",
			content:  `sdkdiag.AppendFromErr(diags, err)`,
			expected: true,
		},
		{
			name:     "convenience function - no migration needed",
			content:  `package main\n\nfunc foo() {}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NeedsMigration(tt.content)
			if result != tt.expected {
				t.Errorf("NeedsMigration() = %v, want %v", result, tt.expected)
			}
		})
	}
}
