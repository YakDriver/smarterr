// types.go
// Core HCL struct definitions for smarterr
package internal

// ...imports...

const (
	// ConfigFileName is the name of the configuration file
	// that contains the configuration for smarterr.
	ConfigFileName = "smarterr.hcl"
)

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

// SmarterrDebug configures internal smarterr debugging logs (not user-facing logs).
type SmarterrDebug struct {
	Output string `hcl:"output,optional"` // Output can be "stdout", "stderr", or a file path.
}

type Config struct {
	SmarterrDebug  *SmarterrDebug `hcl:"smarterr_debug,block"`
	TokenErrorMode string         `hcl:"token_error_mode,optional"` // 	"detailed", "placeholder", "empty" (default: "empty")
	Tokens         []Token        `hcl:"token,block"`
	Hints          []Hint         `hcl:"hint,block"`
	Parameters     []Parameter    `hcl:"parameter,block"`
	StackMatches   []StackMatch   `hcl:"stack_match,block"`
	Templates      []Template     `hcl:"template,block"`
	Transforms     []Transform    `hcl:"transform,block"`
}

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
	Name       string            `hcl:"name,label"`
	Match      map[string]string `hcl:"match"`
	Suggestion string            `hcl:"suggestion"`
}

type StackMatch struct {
	Name        string `hcl:"name,label"`
	CalledFrom  string `hcl:"called_from,optional"`
	CalledAfter string `hcl:"called_after,optional"`
	Display     string `hcl:"display"`
}
