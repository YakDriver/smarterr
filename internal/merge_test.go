package internal

import (
	"reflect"
	"testing"
)

func TestMergeConfigs(t *testing.T) {
	testCases := []struct {
		name        string
		base        Config
		add         Config
		expected    Config
		description string
	}{
		{
			name: "Add new tokens, hints, parameters, stack matches, templates, and transforms",
			base: Config{},
			add: Config{
				Tokens: []Token{
					{Name: "token1", Source: "add"},
					{Name: "token2", Source: "add"},
				},
				Hints: []Hint{
					{Name: "hint1", Suggestion: "value1"},
				},
				Parameters: []Parameter{
					{Name: "param1", Value: "value1"},
					{Name: "param2", Value: "value2"},
				},
				StackMatches: []StackMatch{
					{Name: "match1", CalledFrom: "func1", Display: "Match 1"},
				},
				Templates: []Template{
					{Name: "tmpl1", Format: "{{.foo}}"},
				},
				Transforms: []Transform{
					{Name: "tr1", Steps: []TransformStep{{Type: "lower"}}},
				},
			},
			expected: Config{
				Tokens: []Token{
					{Name: "token1", Source: "add"},
					{Name: "token2", Source: "add"},
				},
				Hints: []Hint{
					{Name: "hint1", Suggestion: "value1"},
				},
				Parameters: []Parameter{
					{Name: "param1", Value: "value1"},
					{Name: "param2", Value: "value2"},
				},
				StackMatches: []StackMatch{
					{Name: "match1", CalledFrom: "func1", Display: "Match 1"},
				},
				Templates: []Template{
					{Name: "tmpl1", Format: "{{.foo}}"},
				},
				Transforms: []Transform{
					{Name: "tr1", Steps: []TransformStep{{Type: "lower"}}},
				},
			},
			description: "Should add all blocks from add to base",
		},
		{
			name: "Overwrite by name for all blocks",
			base: Config{
				Tokens:       []Token{{Name: "token1", Source: "base"}},
				Hints:        []Hint{{Name: "hint1", Suggestion: "base"}},
				Parameters:   []Parameter{{Name: "param1", Value: "base"}},
				StackMatches: []StackMatch{{Name: "match1", CalledFrom: "base", Display: "base"}},
				Templates:    []Template{{Name: "tmpl1", Format: "base"}},
				Transforms:   []Transform{{Name: "tr1", Steps: []TransformStep{{Type: "upper"}}}},
			},
			add: Config{
				Tokens:       []Token{{Name: "token1", Source: "add"}},
				Hints:        []Hint{{Name: "hint1", Suggestion: "add"}},
				Parameters:   []Parameter{{Name: "param1", Value: "add"}},
				StackMatches: []StackMatch{{Name: "match1", CalledFrom: "add", Display: "add"}},
				Templates:    []Template{{Name: "tmpl1", Format: "add"}},
				Transforms:   []Transform{{Name: "tr1", Steps: []TransformStep{{Type: "lower"}}}},
			},
			expected: Config{
				Tokens:       []Token{{Name: "token1", Source: "add"}},
				Hints:        []Hint{{Name: "hint1", Suggestion: "add"}},
				Parameters:   []Parameter{{Name: "param1", Value: "add"}},
				StackMatches: []StackMatch{{Name: "match1", CalledFrom: "add", Display: "add"}},
				Templates:    []Template{{Name: "tmpl1", Format: "add"}},
				Transforms:   []Transform{{Name: "tr1", Steps: []TransformStep{{Type: "lower"}}}},
			},
			description: "Should overwrite by name for all blocks",
		},
		{
			name:        "Merge Smarterr debug and token_error_mode",
			base:        Config{Smarterr: &Smarterr{Debug: false, TokenErrorMode: strPtr("detailed")}},
			add:         Config{Smarterr: &Smarterr{Debug: true, TokenErrorMode: strPtr("placeholder")}},
			expected:    Config{Smarterr: &Smarterr{Debug: true, TokenErrorMode: strPtr("placeholder")}},
			description: "Should overwrite Smarterr debug and token_error_mode",
		},
		{
			name: "No changes when add is empty",
			base: Config{
				Smarterr:     &Smarterr{Debug: true, TokenErrorMode: strPtr("detailed")},
				Tokens:       []Token{{Name: "token1", Source: "base"}},
				Hints:        []Hint{{Name: "hint1", Suggestion: "base"}},
				Parameters:   []Parameter{{Name: "param1", Value: "base"}},
				StackMatches: []StackMatch{{Name: "match1", CalledFrom: "base", Display: "base"}},
				Templates:    []Template{{Name: "tmpl1", Format: "base"}},
				Transforms:   []Transform{{Name: "tr1", Steps: []TransformStep{{Type: "upper"}}}},
			},
			add: Config{},
			expected: Config{
				Smarterr:     &Smarterr{Debug: true, TokenErrorMode: strPtr("detailed")},
				Tokens:       []Token{{Name: "token1", Source: "base"}},
				Hints:        []Hint{{Name: "hint1", Suggestion: "base"}},
				Parameters:   []Parameter{{Name: "param1", Value: "base"}},
				StackMatches: []StackMatch{{Name: "match1", CalledFrom: "base", Display: "base"}},
				Templates:    []Template{{Name: "tmpl1", Format: "base"}},
				Transforms:   []Transform{{Name: "tr1", Steps: []TransformStep{{Type: "upper"}}}},
			},
			description: "Should not change base config when add is empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			base := tc.base
			mergeConfigsPair(&base, &tc.add)
			if !reflect.DeepEqual(base, tc.expected) {
				t.Errorf("Test case '%s' failed: %s\nExpected: %+v\nGot: %+v", tc.name, tc.description, tc.expected, base)
			}
		})
	}
}
