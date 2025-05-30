package internal

import (
	"embed"
)

//go:embed testdata/**/**/smarterr.hcl
//go:embed testdata/**/smarterr.hcl
var testFiles embed.FS

/*
func TestFindConfigPaths(t *testing.T) {
	// Wrap the embedded filesystem in the filesystem WrappedFS implementation
	fsys := &filesystem.WrappedFS{FS: &testFiles}

	// Define test cases
	testCases := []struct {
		name        string
		startDir    string
		expected    []string
		description string
	}{
		{
			name:     "Find all configs starting from subdir",
			startDir: "testdata/project/subdir",
			expected: []string{
				"testdata/internal/smarterr/smarterr.hcl",
				"testdata/project/smarterr.hcl",
				"testdata/project/subdir/smarterr.hcl",
			},
			description: "Should find configs in subdir, parentDir, and global config in internal/smarterr",
		},
		{
			name:     "Find only global config starting from baseDir",
			startDir: "testdata",
			expected: []string{
				"testdata/internal/smarterr/smarterr.hcl",
			},
			description: "Should find only the global config when starting from baseDir",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call findConfigPaths with the specified startDir
			paths, err := findConfigPaths(fsys, tc.startDir, "testdata")
			if err != nil {
				t.Fatalf("findConfigPaths failed: %v", err)
			}

			// Check if the number of paths matches the expected number
			if len(paths) != len(tc.expected) {
				t.Errorf("Test case '%s': Expected %d paths, got %d. Paths: %v",
					tc.name, len(tc.expected), len(paths), paths)
			}

			// Check if the paths match the expected paths
			for i, expectedPath := range tc.expected {
				if i >= len(paths) || paths[i] != expectedPath {
					t.Errorf("Test case '%s': Expected path at index %d to be %s, got %s",
						tc.name, i, expectedPath, paths[i])
				}
			}
		})
	}
}
*/

/*
func TestMergeConfigs(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name        string
		base        Config
		add         Config
		expected    Config
		description string
	}{
		{
			name: "Add new tokens, hints, and parameters",
			base: Config{
				Tokens:     []Token{},
				Hints:      []Hint{},
				Parameters: []Parameter{},
			},
			add: Config{
				Tokens: []Token{
					{Position: 1, Name: "token1", Source: "add"},
					{Position: 2, Name: "token2", Source: "add"},
				},
				Hints: []Hint{
					{Name: "hint1", Suggestion: "value1"},
				},
				Parameters: []Parameter{
					{Name: "param1", Value: "value1"},
					{Name: "param2", Value: "value2"},
				},
			},
			expected: Config{
				Tokens: []Token{
					{Position: 1, Name: "token1", Source: "add"},
					{Position: 2, Name: "token2", Source: "add"},
				},
				Hints: []Hint{
					{Name: "hint1", Suggestion: "value1"},
				},
				Parameters: []Parameter{
					{Name: "param1", Value: "value1"},
					{Name: "param2", Value: "value2"},
				},
			},
			description: "Should add all tokens, hints, and parameters from add to base",
		},
		{
			name: "Overwrite parameters by name",
			base: Config{
				Tokens:     []Token{},
				Hints:      []Hint{},
				Parameters: []Parameter{{Name: "param1", Value: "value1_base"}},
			},
			add: Config{
				Tokens:     []Token{},
				Hints:      []Hint{},
				Parameters: []Parameter{{Name: "param1", Value: "value1_add"}},
			},
			expected: Config{
				Tokens:     []Token{},
				Hints:      []Hint{},
				Parameters: []Parameter{{Name: "param1", Value: "value1_add"}},
			},
			description: "Should overwrite parameters in base by name",
		},
		{
			name: "Add new parameters and keep existing ones",
			base: Config{
				Tokens:     []Token{},
				Hints:      []Hint{},
				Parameters: []Parameter{{Name: "param1", Value: "value1_base"}},
			},
			add: Config{
				Tokens:     []Token{},
				Hints:      []Hint{},
				Parameters: []Parameter{{Name: "param2", Value: "value2_add"}},
			},
			expected: Config{
				Tokens: []Token{},
				Hints:  []Hint{},
				Parameters: []Parameter{
					{Name: "param1", Value: "value1_base"},
					{Name: "param2", Value: "value2_add"},
				},
			},
			description: "Should add new parameters from add and keep existing ones in base",
		},
		{
			name: "No changes when add is empty",
			base: Config{
				Tokens: []Token{
					{Position: 1, Name: "token1", Source: "base"},
				},
				Hints: []Hint{
					{Name: "hint1", Suggestion: "value1_base"},
				},
				Parameters: []Parameter{
					{Name: "param1", Value: "value1_base"},
				},
			},
			add: Config{
				Tokens:     []Token{},
				Hints:      []Hint{},
				Parameters: []Parameter{},
			},
			expected: Config{
				Tokens: []Token{
					{Position: 1, Name: "token1", Source: "base"},
				},
				Hints: []Hint{
					{Name: "hint1", Suggestion: "value1_base"},
				},
				Parameters: []Parameter{
					{Name: "param1", Value: "value1_base"},
				},
			},
			description: "Should make no changes when add is empty",
		},
		{
			name: "Overwrite tokens, hints, and parameters",
			base: Config{
				Tokens: []Token{
					{Position: 1, Name: "token1", Source: "base"},
				},
				Hints: []Hint{
					{Name: "hint1", Suggestion: "value1_base"},
				},
				Parameters: []Parameter{
					{Name: "param1", Value: "value1_base"},
				},
			},
			add: Config{
				Tokens: []Token{
					{Position: 1, Name: "token1_new", Source: "add"},
				},
				Hints: []Hint{
					{Name: "hint1", Suggestion: "value1_add"},
				},
				Parameters: []Parameter{
					{Name: "param1", Value: "value1_add"},
				},
			},
			expected: Config{
				Tokens: []Token{
					{Position: 1, Name: "token1_new", Source: "add"},
				},
				Hints: []Hint{
					{Name: "hint1", Suggestion: "value1_add"},
				},
				Parameters: []Parameter{
					{Name: "param1", Value: "value1_add"},
				},
			},
			description: "Should overwrite tokens, hints, and parameters in base with those from add",
		},
		{
			name: "Add new tokens, hints, parameters, and stack matches",
			base: Config{
				Tokens:       []Token{},
				Hints:        []Hint{},
				Parameters:   []Parameter{},
				StackMatches: []StackMatch{},
			},
			add: Config{
				Tokens: []Token{
					{Position: 1, Name: "token1", Source: "add"},
					{Position: 2, Name: "token2", Source: "add"},
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
					{Name: "match2", CalledFrom: "func2", Display: "Match 2"},
				},
			},
			expected: Config{
				Tokens: []Token{
					{Position: 1, Name: "token1", Source: "add"},
					{Position: 2, Name: "token2", Source: "add"},
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
					{Name: "match2", CalledFrom: "func2", Display: "Match 2"},
				},
			},
			description: "Should add all tokens, hints, parameters, and stack matches from add to base",
		},
		{
			name: "Overwrite stack matches by name",
			base: Config{
				StackMatches: []StackMatch{
					{Name: "match1", CalledFrom: "func1_base", Display: "Match 1 Base"},
				},
			},
			add: Config{
				StackMatches: []StackMatch{
					{Name: "match1", CalledFrom: "func1_add", Display: "Match 1 Add"},
				},
			},
			expected: Config{
				StackMatches: []StackMatch{
					{Name: "match1", CalledFrom: "func1_add", Display: "Match 1 Add"},
				},
			},
			description: "Should overwrite stack matches in base by name",
		},
		{
			name: "Add new stack matches and keep existing ones",
			base: Config{
				StackMatches: []StackMatch{
					{Name: "match1", CalledFrom: "func1_base", Display: "Match 1 Base"},
				},
			},
			add: Config{
				StackMatches: []StackMatch{
					{Name: "match2", CalledFrom: "func2_add", Display: "Match 2 Add"},
				},
			},
			expected: Config{
				StackMatches: []StackMatch{
					{Name: "match1", CalledFrom: "func1_base", Display: "Match 1 Base"},
					{Name: "match2", CalledFrom: "func2_add", Display: "Match 2 Add"},
				},
			},
			description: "Should add new stack matches from add and keep existing ones in base",
		},
		{
			name: "No changes when add is empty for stack matches",
			base: Config{
				StackMatches: []StackMatch{
					{Name: "match1", CalledFrom: "func1_base", Display: "Match 1 Base"},
				},
			},
			add: Config{
				StackMatches: []StackMatch{},
			},
			expected: Config{
				StackMatches: []StackMatch{
					{Name: "match1", CalledFrom: "func1_base", Display: "Match 1 Base"},
				},
			},
			description: "Should make no changes when add is empty for stack matches",
		},
		{
			name: "Merge LogOutput, LogLevel, and Fallback",
			base: Config{
				LogOutput: "stdout",
				LogLevel:  "info",
				Fallback:  "basic",
			},
			add: Config{
				LogOutput: "stderr",
				LogLevel:  "debug",
				Fallback:  "detailed",
			},
			expected: Config{
				LogOutput: "stderr",
				LogLevel:  "debug",
				Fallback:  "detailed",
			},
			description: "Should overwrite Debug, LogOutput, LogLevel, and Fallback in base with those from add",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Copy base to avoid modifying the original
			base := tc.base

			// Call mergeConfigs
			mergeConfigs(&base, &tc.add)

			// Verify the result
			if !reflect.DeepEqual(base, tc.expected) {
				t.Errorf("Test case '%s' failed: %s\nExpected: %+v\nGot: %+v",
					tc.name, tc.description, tc.expected, base)
			}
		})
	}
}
*/

/*
func TestLoadConfig(t *testing.T) {
	// Wrap the embedded filesystem in the smarterr EmbeddedFS implementation
	fsys := &filesystem.WrappedFS{FS: &testFiles}

	// Define test cases
	testCases := []struct {
		name        string
		startDir    string
		expected    Config
		description string
	}{
		{
			name:     "Merge all configs starting from subdir",
			startDir: "testdata/project/subdir",
			expected: Config{
				Tokens: []Token{
					{Position: 3, Name: "example", Source: "local"},
					{Position: 4, Name: "param_token", Source: "parameter", Parameter: stringPtr("param1")},
				},
				Hints: []Hint{
					{
						Name: "hint1",
						Match: map[string]string{
							"error_detail": "global match",
						},
						Suggestion: "global hint",
					},
					{
						Name: "hint2",
						Match: map[string]string{
							"error_detail": "parent match",
						},
						Suggestion: "parent hint",
					},
					{
						Name: "hint3",
						Match: map[string]string{
							"error_detail": "local match",
						},
						Suggestion: "local hint",
					},
				},
				Parameters: []Parameter{
					{Name: "param1", Value: "parent_value"},
					{Name: "param2", Value: "local_value"},
				},
			},
			description: "Should merge configs from local, parent, and global with proper precedence",
		},
		{
			name:     "Include only global config starting from baseDir",
			startDir: "testdata",
			expected: Config{
				Tokens: []Token{
					{Position: 1, Name: "example", Source: "hint"},
				},
				Hints: []Hint{
					{
						Name: "hint1",
						Match: map[string]string{
							"error_detail": "global match",
						},
						Suggestion: "global hint",
					},
				},
				Parameters: []Parameter{
					{Name: "param1", Value: "global_value"},
				},
			},
			description: "Should include only the global config when starting from baseDir",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call LoadConfig with the specified startDir
			config, err := LoadConfig(fsys, tc.startDir, "testdata")
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}

			// Verify the result
			if !reflect.DeepEqual(config, &tc.expected) {
				t.Errorf("Test case '%s' failed: %s\nExpected: %+v\nGot: %+v",
					tc.name, tc.description, tc.expected, config)
			}
		})
	}
}
*/
