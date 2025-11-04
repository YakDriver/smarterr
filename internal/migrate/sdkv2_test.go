package migrate

import "testing"

func TestCreateSDKv2Patterns(t *testing.T) {
	patterns := CreateSDKv2Patterns()

	if patterns.Name != "SDKv2Patterns" {
		t.Errorf("Expected name 'SDKv2Patterns', got %s", patterns.Name)
	}

	if patterns.Order != 4 {
		t.Errorf("Expected order 4, got %d", patterns.Order)
	}

	if len(patterns.Patterns) == 0 {
		t.Error("Expected patterns to be non-empty")
	}
}

func TestSDKv2_AppendErrorf(t *testing.T) {
	migrator := NewMigrator(MigratorOptions{})

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "sdkdiag.AppendErrorf with message and error",
			input:    "\treturn sdkdiag.AppendErrorf(diags, \"reading VPC: %s\", err)\n",
			expected: "\treturn smerr.Append(ctx, diags, err)\n",
		},
		{
			name:     "sdkdiag.AppendErrorf with id and error",
			input:    "\treturn sdkdiag.AppendErrorf(diags, \"reading VPC (%s): %s\", id, err)\n",
			expected: "\treturn smerr.Append(ctx, diags, err, smerr.ID, id)\n",
		},
		{
			name:     "sdkdiag.AppendErrorf with complex message",
			input:    "\treturn sdkdiag.AppendErrorf(diags, \"waiting for VPC (%s) creation: %s\", aws.ToString(output.VpcId), err)\n",
			expected: "\treturn smerr.Append(ctx, diags, err, smerr.ID, aws.ToString(output.VpcId))\n",
		},
		{
			name: "sdkdiag.AppendErrorf should not corrupt subsequent d.Set calls",
			input: `	if err := d.Set("lambda_config", flattenLambdaDataSourceConfig(dataSource.LambdaConfig)); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting lambda_config: %s", err)
	}
	d.Set(names.AttrName, dataSource.Name)`,
			expected: `	if err := d.Set("lambda_config", flattenLambdaDataSourceConfig(dataSource.LambdaConfig)); err != nil {
		return smerr.Append(ctx, diags, err)
	}
	d.Set(names.AttrName, dataSource.Name)`,
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
