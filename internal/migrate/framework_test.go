package migrate

import "testing"

func TestCreateProblemStandardMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "single-line create.ProblemStandardMessage",
			input: `		response.Diagnostics.AddError(create.ProblemStandardMessage(names.AppSync, create.ErrActionCreating, "test", "id", err), err.Error())`,
			expected: `		smerr.AddError(ctx, &response.Diagnostics, err)`,
		},
		{
			name: "multi-line create.ProblemStandardMessage with err.Error()",
			input: `	response.Diagnostics.AddError(
		create.ProblemStandardMessage(names.AppSync, create.ErrActionCreating, resNameSourceAPIAssociation, plan.MergedAPIID.String(), err),
		err.Error(),
	)`,
			expected: `	smerr.AddError(ctx, &response.Diagnostics, err)`,
		},
		{
			name: "multi-line create.ProblemStandardMessage with errors.New",
			input: `	response.Diagnostics.AddError(
		create.ProblemStandardMessage(names.AppSync, create.ErrActionCreating, resNameSourceAPIAssociation, plan.MergedAPIID.String(), nil),
		errors.New("empty output").Error(),
	)`,
			expected: `	smerr.AddError(ctx, &response.Diagnostics, errors.New("empty output"))`,
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
