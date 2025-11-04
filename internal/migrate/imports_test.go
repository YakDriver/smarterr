package migrate

import (
	"strings"
	"testing"
)

func TestImportManager_AddRequiredImports(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // Expected imports to be present
	}{
		{
			name: "adds missing imports to simple import block",
			input: `package main

import (
	"context"
	"fmt"
)

func main() {}`,
			expected: []string{
				`"github.com/YakDriver/smarterr"`,
				`"github.com/hashicorp/terraform-provider-aws/internal/smerr"`,
			},
		},
		{
			name: "adds imports after existing imports",
			input: `package main

import (
	"context"
	
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func main() {}`,
			expected: []string{
				`"github.com/YakDriver/smarterr"`,
				`"github.com/hashicorp/terraform-provider-aws/internal/smerr"`,
			},
		},
		{
			name: "skips existing imports",
			input: `package main

import (
	"fmt"
	"github.com/YakDriver/smarterr"
	"github.com/hashicorp/terraform-provider-aws/internal/smerr"
)

func main() {}`,
			expected: []string{
				`"github.com/YakDriver/smarterr"`,
				`"github.com/hashicorp/terraform-provider-aws/internal/smerr"`,
			},
		},
		{
			name: "adds only smarterr if smerr exists",
			input: `package main

import (
	"context"
	"github.com/hashicorp/terraform-provider-aws/internal/smerr"
)

func main() {}`,
			expected: []string{
				`"github.com/YakDriver/smarterr"`,
				`"github.com/hashicorp/terraform-provider-aws/internal/smerr"`,
			},
		},
		{
			name: "adds only smerr if smarterr exists",
			input: `package main

import (
	"context"
	"github.com/YakDriver/smarterr"
)

func main() {}`,
			expected: []string{
				`"github.com/YakDriver/smarterr"`,
				`"github.com/hashicorp/terraform-provider-aws/internal/smerr"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewImportManager(tt.input)
			result := im.AddRequiredImports()

			for _, expectedImport := range tt.expected {
				if !strings.Contains(result, expectedImport) {
					t.Errorf("Expected import %s not found in result", expectedImport)
				}
			}
		})
	}
}

func TestImportManager_GetRetryPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "no conflict - returns retry",
			input: `package main

import (
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
)`,
			expected: "retry",
		},
		{
			name: "conflict detected - returns intretry",
			input: `package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)`,
			expected: "intretry",
		},
		{
			name: "both imports present - returns retry",
			input: `package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
)`,
			expected: "retry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewImportManager(tt.input)
			result := im.GetRetryPrefix()

			if result != tt.expected {
				t.Errorf("GetRetryPrefix() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestImportManager_HasConflictingRetryImport(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name: "has conflicting import",
			input: `package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)`,
			expected: true,
		},
		{
			name: "no conflicting import",
			input: `package main

import (
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
)`,
			expected: false,
		},
		{
			name: "both imports present - no conflict",
			input: `package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
)`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewImportManager(tt.input)
			result := im.HasConflictingRetryImport()

			if result != tt.expected {
				t.Errorf("HasConflictingRetryImport() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestImportManager_AddAliasedRetryImport(t *testing.T) {
	input := `package main

import (
	"fmt"
)

func main() {}`

	im := NewImportManager(input)
	result := im.AddAliasedRetryImport()

	expectedImport := `intretry "github.com/hashicorp/terraform-provider-aws/internal/retry"`
	if !strings.Contains(result, expectedImport) {
		t.Errorf("Expected aliased import %s not found in result", expectedImport)
	}
}

func TestCreateImportPatterns(t *testing.T) {
	patterns := CreateImportPatterns()

	if patterns.Name != "ImportPatterns" {
		t.Errorf("Expected name 'ImportPatterns', got %s", patterns.Name)
	}

	if patterns.Order != 0 {
		t.Errorf("Expected order 0, got %d", patterns.Order)
	}

	if len(patterns.Patterns) == 0 {
		t.Error("Expected patterns to be non-empty")
	}
}
