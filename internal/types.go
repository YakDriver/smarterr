// types.go
// Core HCL struct definitions for smarterr
package internal

// ...imports...

const (
	// ConfigFileName is the name of the configuration file
	// that contains the configuration for smarterr.
	ConfigFileName = "smarterr.hcl"
)

// Config represents the top-level configuration for smarterr.
type Config struct {
	Smarterr     *Smarterr    `hcl:"smarterr,block"`
	Tokens       []Token      `hcl:"token,block"`
	Hints        []Hint       `hcl:"hint,block"`
	Parameters   []Parameter  `hcl:"parameter,block"`
	StackMatches []StackMatch `hcl:"stack_match,block"`
	Templates    []Template   `hcl:"template,block"`
	Transforms   []Transform  `hcl:"transform,block"`
}

// Smarterr represents settings for how smarterr works such as debugging, token error mode, etc.
type Smarterr struct {
	Debug          bool    `hcl:"debug,optional"`
	TokenErrorMode string  `hcl:"token_error_mode,optional"` // "detailed", "placeholder", "empty" (default: "empty")
	HintJoinChar   *string `hcl:"hint_join_char,optional"`
	HintMatchMode  *string `hcl:"hint_match_mode,optional"` // "all" (default), "first"
}

// Template represents a named text/template for formatting error messages or diagnostics.
type Template struct {
	Name   string `hcl:"name,label"`
	Format string `hcl:"format"`
}

type TransformStep struct {
	Type    string  `hcl:"type,label"`
	Value   *string `hcl:"value,optional"`
	Regex   *string `hcl:"regex,optional"`
	With    *string `hcl:"with,optional"`
	Recurse *bool   `hcl:"recurse,optional"`
}

type Transform struct {
	Name  string          `hcl:"name,label"`
	Steps []TransformStep `hcl:"step,block"`
}

// Token represents a token in the configuration, which can be used for error message formatting.
type Token struct {
	Name         string   `hcl:"name,label"`
	Source       string   `hcl:"source,optional"`
	Parameter    *string  `hcl:"parameter,optional"`
	StackMatches []string `hcl:"stack_matches,optional"`
	Arg          *string  `hcl:"arg,optional"`
	Context      *string  `hcl:"context,optional"`
	Pattern      *string  `hcl:"pattern,optional"`
	Replace      *string  `hcl:"replace,optional"`
	Transforms   []string `hcl:"transforms,optional"`
}

type Parameter struct {
	Name  string `hcl:"name,label"`
	Value string `hcl:"value,attr"`
}

type Hint struct {
	Name          string  `hcl:"name,label"`
	ErrorContains *string `hcl:"error_contains,optional"`
	RegexMatch    *string `hcl:"regex_match,optional"`
	Suggestion    string  `hcl:"suggestion"`
}

type StackMatch struct {
	Name        string `hcl:"name,label"`
	CalledFrom  string `hcl:"called_from,optional"`
	CalledAfter string `hcl:"called_after,optional"`
	Display     string `hcl:"display"`
}
