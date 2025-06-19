package internal

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func strPtr(s string) *string { return &s }

func TestParseKeyvals(t *testing.T) {
	tests := []struct {
		name  string
		input []any
		want  map[string]any
	}{
		{
			name:  "Valid key-value pairs",
			input: []any{"id", "rds", "service", "Provider"},
			want:  map[string]any{"id": "rds", "service": "Provider"},
		},
		{
			name:  "Odd number of arguments",
			input: []any{"id", "rds", "service"},
			want:  map[string]any{"id": "rds"},
		},
		{
			name:  "Non-string key",
			input: []any{123, "rds", "service", "Provider"},
			want:  map[string]any{},
		},
		{
			name:  "Empty input",
			input: []any{},
			want:  map[string]any{},
		},
		{
			name:  "Duplicate keys",
			input: []any{"id", "rds", "id", "new_rds"},
			want:  map[string]any{"id": "new_rds"},
		},
		{
			name:  "Nil value",
			input: []any{"id", nil, "service", "Provider"},
			want:  map[string]any{"id": nil, "service": "Provider"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call parseKeyvals
			got := parseKeyvals(context.Background(), tt.input...)

			// Verify the result
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseKeyvals() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenResolve_BasicSources(t *testing.T) {
	tests := []struct {
		name    string
		token   Token
		ctx     context.Context
		runtime *Runtime
		want    string
	}{
		{
			name:    "parameter found",
			token:   Token{Source: "parameter", Parameter: stringPtr("foo")},
			ctx:     context.Background(),
			runtime: NewRuntime(context.Background(), &Config{Parameters: []Parameter{{Name: "foo", Value: "bar"}}}, nil, nil),
			want:    "bar",
		},
		{
			name:    "parameter not found",
			token:   Token{Source: "parameter", Parameter: stringPtr("baz")},
			ctx:     context.Background(),
			runtime: NewRuntime(context.Background(), &Config{Parameters: []Parameter{{Name: "foo", Value: "bar"}}}, nil, nil),
			want:    "",
		},
		{
			name:    "context found",
			token:   Token{Source: "context", Context: stringPtr("key")},
			ctx:     context.WithValue(context.Background(), "key", "val"),
			runtime: NewRuntime(context.Background(), &Config{}, nil, nil),
			want:    "val",
		},
		{
			name:    "context not found",
			token:   Token{Source: "context", Context: stringPtr("missing")},
			ctx:     context.WithValue(context.Background(), "key", "val"),
			runtime: NewRuntime(context.Background(), &Config{}, nil, nil),
			want:    "",
		},
		{
			name:    "error present",
			token:   Token{Source: "error"},
			ctx:     context.Background(),
			runtime: NewRuntime(context.Background(), &Config{}, fmt.Errorf("fail"), nil),
			want:    "fail",
		},
		{
			name:    "error absent",
			token:   Token{Source: "error"},
			ctx:     context.Background(),
			runtime: NewRuntime(context.Background(), &Config{}, nil, nil),
			want:    "",
		},
		{
			name:    "arg found",
			token:   Token{Source: "arg", Arg: stringPtr("foo")},
			ctx:     context.Background(),
			runtime: NewRuntime(context.Background(), &Config{}, nil, "foo", "bar"),
			want:    "bar",
		},
		{
			name:    "arg not found",
			token:   Token{Source: "arg", Arg: stringPtr("baz")},
			ctx:     context.Background(),
			runtime: NewRuntime(context.Background(), &Config{}, nil, "foo", "bar"),
			want:    "",
		},
		{
			name:    "unknown source fallback",
			token:   Token{Source: "unknown"},
			ctx:     context.Background(),
			runtime: NewRuntime(context.Background(), &Config{}, nil, nil),
			want:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.token.Resolve(tc.ctx, tc.runtime)
			if got != tc.want {
				t.Errorf("Resolve() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRuntime_BuildTokenValueMap(t *testing.T) {
	ctx := context.WithValue(context.Background(), "ctxKey", "ctxVal")
	cfg := &Config{
		Parameters: []Parameter{{Name: "param1", Value: "val1"}},
		Tokens: []Token{
			{Name: "param_token", Source: "parameter", Parameter: stringPtr("param1")},
			{Name: "ctx_token", Source: "context", Context: stringPtr("ctxKey")},
			{Name: "error_token", Source: "error"},
			{Name: "arg_token", Source: "arg", Arg: stringPtr("foo")},
			{Name: "missing_param", Source: "parameter", Parameter: stringPtr("notfound")},
			{Name: "missing_ctx", Source: "context", Context: stringPtr("notfound")},
			{Name: "missing_arg", Source: "arg", Arg: stringPtr("notfound")},
		},
	}
	err := fmt.Errorf("errVal")
	rt := NewRuntime(context.Background(), cfg, err, "foo", "bar")
	got := rt.BuildTokenValueMap(ctx)
	want := map[string]any{
		"param_token":   "val1",
		"ctx_token":     "ctxVal",
		"error_token":   "errVal",
		"arg_token":     "bar",
		"missing_param": "",
		"missing_ctx":   "",
		"missing_arg":   "",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("BuildTokenValueMap()[%q] = %v, want %v", k, got[k], v)
		}
	}
}

func TestTokenResolve_WithTransforms(t *testing.T) {
	toLower := "lower"
	stripPrefix := "strip_prefix"
	fixSpace := "fix_space"
	prefixVal := "PRE_"
	cfg := &Config{
		Parameters: []Parameter{{Name: "p", Value: "PRE_  Foo   Bar  "}},
		Transforms: []Transform{
			{
				Name:  stripPrefix,
				Steps: []TransformStep{{Type: "strip_prefix", Value: &prefixVal}},
			},
			{
				Name:  fixSpace,
				Steps: []TransformStep{{Type: "fix_space"}},
			},
			{
				Name:  toLower,
				Steps: []TransformStep{{Type: "lower"}},
			},
		},
	}
	token := Token{
		Name:       "t",
		Source:     "parameter",
		Parameter:  stringPtr("p"),
		Transforms: []string{stripPrefix, fixSpace, toLower},
	}
	rt := NewRuntime(context.Background(), cfg, nil, nil)
	got := token.Resolve(context.Background(), rt)
	want := "foo bar"
	if got != want {
		t.Errorf("Resolve() with transforms = %q, want %q", got, want)
	}
}

func TestTokenResolve_CallStackSource(t *testing.T) {
	// Setup a StackMatch that matches this test function
	stackMatchName := "testMatch"
	calledFromRegex := "TestTokenResolve_CallStackSource"
	displayVal := "matched!"
	cfg := &Config{
		StackMatches: []StackMatch{{
			Name:       stackMatchName,
			CalledFrom: calledFromRegex,
			Display:    displayVal,
		}},
		Tokens: []Token{{
			Name:         "stack_token",
			Source:       "call_stack",
			StackMatches: []string{stackMatchName},
		}},
	}
	rt := NewRuntime(context.Background(), cfg, nil, nil)
	token := cfg.Tokens[0]
	got := token.Resolve(context.Background(), rt)
	if got != displayVal {
		t.Errorf("Resolve() call_stack = %q, want %q", got, displayVal)
	}
}

func TestConfig_RenderTemplate_BasicAndFallback(t *testing.T) {
	cfg := &Config{
		Templates: []Template{{
			Name:   "hello",
			Format: "Hello, {{.name}}! Your id is {{.id}}.",
		}},
		Smarterr: &Smarterr{TokenErrorMode: strPtr("placeholder")},
	}
	values := map[string]any{"name": "Alice"} // id is missing
	out, err := cfg.RenderTemplate(context.Background(), "hello", values)
	if err != nil {
		t.Fatalf("RenderTemplate error: %v", err)
	}
	want := "Hello, Alice! Your id is <id>."
	if out != want {
		t.Errorf("RenderTemplate output = %q, want %q", out, want)
	}
}

func TestConfig_RenderTemplate_AllVarsPresent(t *testing.T) {
	cfg := &Config{
		Templates: []Template{{
			Name:   "bye",
			Format: "Bye, {{.name}}! See you at {{.place}}.",
		}},
	}
	values := map[string]any{"name": "Bob", "place": "the park"}
	out, err := cfg.RenderTemplate(context.Background(), "bye", values)
	if err != nil {
		t.Fatalf("RenderTemplate error: %v", err)
	}
	want := "Bye, Bob! See you at the park."
	if out != want {
		t.Errorf("RenderTemplate output = %q, want %q", out, want)
	}
}

func TestConfig_RenderTemplate_TemplateNotFound(t *testing.T) {
	cfg := &Config{
		Templates: []Template{{Name: "exists", Format: "Hi"}},
	}
	_, err := cfg.RenderTemplate(context.Background(), "missing", map[string]any{})
	if err == nil || err.Error() != "template \"missing\" not found" {
		t.Errorf("Expected not found error, got: %v", err)
	}
}

func TestConfig_RenderTemplate_SyntaxError(t *testing.T) {
	cfg := &Config{
		Templates: []Template{
			{Name: "bad", Format: "{{.name"},
		},
	}
	_, err := cfg.RenderTemplate(context.Background(), "bad", map[string]any{"name": "X"})
	if err == nil {
		t.Errorf("Expected syntax error, got nil")
	}
}

func TestCollectTemplateVariables(t *testing.T) {
	tmpl, err := template.New("vars").Parse("Hello, {{.foo}} and {{.bar}}! {{if .baz}}{{.baz}}{{end}}")
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}
	vars := CollectTemplateVariables(tmpl)
	want := map[string]bool{"foo": true, "bar": true, "baz": true}
	for _, v := range vars {
		if !want[v] {
			t.Errorf("collectTemplateVariables: unexpected var %q", v)
		}
		delete(want, v)
	}
	for v := range want {
		t.Errorf("collectTemplateVariables: missing var %q", v)
	}
}

func TestTokenResolve_HintsSource(t *testing.T) {
	errStr := "operation error RDS: ModifyDBCluster, https response error StatusCode: 400, RequestID: abc-123, api error InvalidParameterCombination: You can't change your Performance Insights KMS key."
	contains := "can't change your Performance Insights KMS key"
	regex := "ModifyDBCluster.*InvalidParameterCombination"
	cfg := &Config{
		Hints: []Hint{
			{Name: "abc", ErrorContains: &contains, Suggestion: "Make sure you..."},
			{Name: "bcd", RegexMatch: &regex, Suggestion: "Your parameters aren't right..."},
		},
		Tokens: []Token{
			{Name: "suggest", Source: "hints"},
		},
	}
	rt := NewRuntime(context.Background(), cfg, fmt.Errorf("%s", errStr), nil)
	ctx := context.Background()
	val := cfg.Tokens[0].Resolve(ctx, rt)
	want := "Make sure you...\nYour parameters aren't right..."
	if val != want {
		t.Errorf("expected suggestions:\n%q\ngot:\n%q", want, val)
	}
}

func TestProcessStackMatches_CalledFromPreference(t *testing.T) {
	matches := []StackMatch{
		{Name: "create", CalledFrom: "resource[a-zA-Z0-9]*Create", Display: "creating"},
		{Name: "read", CalledFrom: "resource[a-zA-Z0-9]*Read", Display: "reading"},
		{Name: "wait", CalledFrom: "wait.*", Display: "waiting during operation"},
		{Name: "find", CalledFrom: "find.*", Display: "finding during operation"},
		{Name: "set", CalledFrom: "Set", Display: "setting during operation"},
	}

	tests := []struct {
		name   string
		frames []runtime.Frame
		want   string
	}{
		{
			name:   "match wait subaction",
			frames: []runtime.Frame{{Function: "waitForSomething"}, {Function: "resourceFooCreate"}},
			want:   "waiting during operation",
		},
		{
			name:   "match find subaction",
			frames: []runtime.Frame{{Function: "findBar"}, {Function: "resourceFooRead"}},
			want:   "finding during operation",
		},
		{
			name:   "match set subaction",
			frames: []runtime.Frame{{Function: "Set"}, {Function: "resourceBarRead"}},
			want:   "setting during operation",
		},
		{
			name:   "fallback to create",
			frames: []runtime.Frame{{Function: "resourceFooCreate"}},
			want:   "creating",
		},
		{
			name:   "fallback to read",
			frames: []runtime.Frame{{Function: "resourceFooRead"}},
			want:   "reading",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			display, err := processStackMatches(matches, tc.frames)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if display != tc.want {
				t.Errorf("got %q, want %q", display, tc.want)
			}
		})
	}
}

type mockDiag struct{}
type Severity int

func (Severity) String() string {
	return SeverityError
}

func (mockDiag) Summary() string            { return "Something went wrong" }
func (mockDiag) Detail() string             { return "A detailed explanation" }
func (mockDiag) Severity() diag.Severity    { return 1 }
func (mockDiag) Equal(diag.Diagnostic) bool { return true }

func TestTokenResolve_DiagnosticSource(t *testing.T) {
	cfg := &Config{
		Transforms: []Transform{
			{
				Name:  "upper",
				Steps: []TransformStep{{Type: "upper"}},
			},
			{
				Name:  "lower",
				Steps: []TransformStep{{Type: "lower"}},
			},
		},
	}
	token := Token{
		Name:   "diag",
		Source: "diagnostic",
		FieldTransforms: map[string][]string{
			"summary": {"upper"},
			"detail":  {"lower"},
		},
	}
	rt := NewRuntimeForDiagnostic(context.Background(), cfg, mockDiag{})
	ctx := context.Background()
	val := token.Resolve(ctx, rt)
	diagMap, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("diagnostic token did not return map[string]any, got %T", val)
	}
	if diagMap["summary"] != "SOMETHING WENT WRONG" {
		t.Errorf("summary transform failed: got %q, want %q", diagMap["summary"], "SOMETHING WENT WRONG")
	}
	if diagMap["detail"] != "a detailed explanation" {
		t.Errorf("detail transform failed: got %q, want %q", diagMap["detail"], "a detailed explanation")
	}
	if diagMap["severity"] != "Error" {
		t.Errorf("severity should be unchanged: got %q, want %q", diagMap["severity"], "Error")
	}
}
