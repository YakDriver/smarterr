package smarterr

import (
	"context"
	"testing"

	fwdiag "github.com/hashicorp/terraform-plugin-framework/diag"
	sdkdiag "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func TestAddOne_PreservesDiagnosticSeverity(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		input    fwdiag.Diagnostic
		expected string
	}{
		{
			name:     "warning diagnostic preserved",
			input:    fwdiag.NewWarningDiagnostic("test warning", "warning detail"),
			expected: SeverityWarning,
		},
		{
			name:     "error diagnostic preserved",
			input:    fwdiag.NewErrorDiagnostic("test error", "error detail"),
			expected: SeverityError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var existing fwdiag.Diagnostics

			AddOne(ctx, &existing, tt.input)

			if len(existing) != 1 {
				t.Fatalf("expected 1 diagnostic, got %d", len(existing))
			}

			actualSeverity := existing[0].Severity().String()
			if actualSeverity != tt.expected {
				t.Errorf("expected severity %s, got %s", tt.expected, actualSeverity)
			}
		})
	}
}

func TestAddEnrich_PreservesDiagnosticSeverity(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		input    fwdiag.Diagnostics
		expected string
	}{
		{
			name:     "warning diagnostic preserved",
			input:    fwdiag.Diagnostics{fwdiag.NewWarningDiagnostic("test warning", "warning detail")},
			expected: SeverityWarning,
		},
		{
			name:     "error diagnostic preserved",
			input:    fwdiag.Diagnostics{fwdiag.NewErrorDiagnostic("test error", "error detail")},
			expected: SeverityError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var existing fwdiag.Diagnostics

			AddEnrich(ctx, &existing, tt.input)

			if len(existing) != 1 {
				t.Fatalf("expected 1 diagnostic, got %d", len(existing))
			}

			actualSeverity := existing[0].Severity().String()
			if actualSeverity != tt.expected {
				t.Errorf("expected severity %s, got %s", tt.expected, actualSeverity)
			}
		})
	}
}

func TestAppendOne_PreservesSDKDiagnosticSeverity(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		input    sdkdiag.Diagnostic
		expected sdkdiag.Severity
	}{
		{
			name: "warning diagnostic preserved",
			input: sdkdiag.Diagnostic{
				Severity: sdkdiag.Warning,
				Summary:  "test warning",
				Detail:   "warning detail",
			},
			expected: sdkdiag.Warning,
		},
		{
			name: "error diagnostic preserved",
			input: sdkdiag.Diagnostic{
				Severity: sdkdiag.Error,
				Summary:  "test error",
				Detail:   "error detail",
			},
			expected: sdkdiag.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var existing sdkdiag.Diagnostics

			result := AppendOne(ctx, existing, tt.input)

			if len(result) != 1 {
				t.Fatalf("expected 1 diagnostic, got %d", len(result))
			}

			if result[0].Severity != tt.expected {
				t.Errorf("expected severity %v, got %v", tt.expected, result[0].Severity)
			}
		})
	}
}

func TestAppendEnrich_PreservesSDKDiagnosticSeverity(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		input    sdkdiag.Diagnostics
		expected sdkdiag.Severity
	}{
		{
			name: "warning diagnostic preserved",
			input: sdkdiag.Diagnostics{{
				Severity: sdkdiag.Warning,
				Summary:  "test warning",
				Detail:   "warning detail",
			}},
			expected: sdkdiag.Warning,
		},
		{
			name: "error diagnostic preserved",
			input: sdkdiag.Diagnostics{{
				Severity: sdkdiag.Error,
				Summary:  "test error",
				Detail:   "error detail",
			}},
			expected: sdkdiag.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var existing sdkdiag.Diagnostics

			result := AppendEnrich(ctx, existing, tt.input)

			if len(result) != 1 {
				t.Fatalf("expected 1 diagnostic, got %d", len(result))
			}

			if result[0].Severity != tt.expected {
				t.Errorf("expected severity %v, got %v", tt.expected, result[0].Severity)
			}
		})
	}
}
