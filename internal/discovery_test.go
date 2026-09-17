package internal

import (
	"context"
	"testing"
	"testing/fstest"
)

func TestLoadConfig_Simple(t *testing.T) {
	// Real-world example:
	// baseDir := "internal"
	// relStackPaths := []string{
	//   "develop/go/src/github.com/YakDriver/smarterr/smarterr.go",
	//   "develop/go/src/github.com/YakDriver/smarterr/smarterr.go",
	//   "develop/go/src/github.com/hashicorp/terraform-provider-aws7/internal/service/cloudwatch/composite_alarm.go",
	//   "develop/go/src/github.com/hashicorp/terraform-provider-aws7/internal/provider/sdkv2/intercept.go",
	//   "develop/go/pkg/mod/github.com/hashicorp/terraform-plugin-sdk/v2@v2.37.0/helper/schema/resource.go",
	// }
	//
	// Config paths in FS (candidates, not all will be used):
	//   service/smarterr.hcl
	//   service/cloudwatch/smarterr.hcl
	//   service/cloudtrail/smarterr.hcl
	//
	// This should result in these configs being used:
	//   service/smarterr.hcl
	//   service/cloudwatch/smarterr.hcl

	// And, NOT used:
	//   service/cloudtrail/smarterr.hcl

	fsys := &WrappedFS{FS: fstest.MapFS{
		"service/smarterr.hcl":         &fstest.MapFile{Data: []byte(`token "foo" {}`)},
		"service/project/smarterr.hcl": &fstest.MapFile{Data: []byte(`token "bar" {}`)},
	}}
	// relStackPaths must contain a path that matches the configDir logic in collectConfigsForStack
	relStackPaths := []string{
		"x/y/z/YakDriver/smarterr/smarterr.go",
		"x/y/z/YakDriver/smarterr/smarterr.go",
		"x/y/z/internal/service/project/smarterr.go",
	}
	baseDir := "internal"

	cfg, err := LoadConfig(context.Background(), fsys, relStackPaths, baseDir)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if len(cfg.Tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(cfg.Tokens))
	}
	names := map[string]bool{}
	for _, tok := range cfg.Tokens {
		names[tok.Name] = true
	}
	if !names["foo"] || !names["bar"] {
		t.Errorf("expected tokens 'foo' and 'bar', got: %v", names)
	}
}

// TestLoadConfig_DotBaseDirCandidate is the regression test for baseDir ".":
// collectRelStackPaths passes absolute frame paths through unchanged, and
// candidate matching uses the bare configDir. Guards against discarding all
// frames in this mode (which would fall back to only the global config).
func TestLoadConfig_DotBaseDirCandidate(t *testing.T) {
	fsys := &WrappedFS{FS: fstest.MapFS{
		"service/smarterr.hcl":       &fstest.MapFile{Data: []byte(`token "base" {}`)},
		"service/amp/smarterr.hcl":   &fstest.MapFile{Data: []byte(`token "amp" {}`)},
		"service/other/smarterr.hcl": &fstest.MapFile{Data: []byte(`token "other" {}`)},
	}}
	relStackPaths := []string{
		"/abs/proj/service/amp/anomaly_detector_list.go",
		"/usr/local/go/src/runtime/proc.go",
	}

	cfg, err := LoadConfig(context.Background(), fsys, relStackPaths, ".")
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	names := map[string]bool{}
	for _, tok := range cfg.Tokens {
		names[tok.Name] = true
	}
	if !names["base"] || !names["amp"] {
		t.Errorf("expected 'base' and 'amp' tokens discovered for baseDir \".\", got: %v", names)
	}
	if names["other"] {
		t.Errorf("token 'other' should not be discovered, got: %v", names)
	}
}

// TestLoadConfig_PrefixCollisionSiblings guards against one service's config
// bleeding into another whose name shares a prefix (e.g. amp/amplify,
// acm/acmpca). The candidate configDir must match a full path segment.
func TestLoadConfig_PrefixCollisionSiblings(t *testing.T) {
	fsys := &WrappedFS{FS: fstest.MapFS{
		"service/smarterr.hcl":         &fstest.MapFile{Data: []byte(`token "base" {}`)},
		"service/amp/smarterr.hcl":     &fstest.MapFile{Data: []byte(`token "amp" {}`)},
		"service/amplify/smarterr.hcl": &fstest.MapFile{Data: []byte(`token "amplify" {}`)},
		"service/acm/smarterr.hcl":     &fstest.MapFile{Data: []byte(`token "acm" {}`)},
		"service/acmpca/smarterr.hcl":  &fstest.MapFile{Data: []byte(`token "acmpca" {}`)},
	}}

	tests := []struct {
		name      string
		frame     string
		want      string // service token that must be present
		mustNotBe string // sibling token that must be absent
	}{
		{"amplify frame does not pull amp", "x/y/z/internal/service/amplify/resource_app.go", "amplify", "amp"},
		{"amp frame does not pull amplify", "x/y/z/internal/service/amp/workspace.go", "amp", "amplify"},
		{"acmpca frame does not pull acm", "x/y/z/internal/service/acmpca/certificate_authority.go", "acmpca", "acm"},
		{"acm frame does not pull acmpca", "x/y/z/internal/service/acm/certificate.go", "acm", "acmpca"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := LoadConfig(context.Background(), fsys, []string{tc.frame}, "internal")
			if err != nil {
				t.Fatalf("LoadConfig error: %v", err)
			}
			names := map[string]bool{}
			for _, tok := range cfg.Tokens {
				names[tok.Name] = true
			}
			if !names["base"] {
				t.Errorf("expected shared 'base' config, got: %v", names)
			}
			if !names[tc.want] {
				t.Errorf("expected %q config for frame %q, got: %v", tc.want, tc.frame, names)
			}
			if names[tc.mustNotBe] {
				t.Errorf("%q config must not bleed into %q; got: %v", tc.mustNotBe, tc.want, names)
			}
		})
	}
}

// TestLoadConfig_LeftBoundary guards the left edge of the segment match: a
// directory that merely ends with the configured baseDir/configDir name (e.g.
// "notinternal", "notservice") must not be treated as a frame under the root.
func TestLoadConfig_LeftBoundary(t *testing.T) {
	fsys := &WrappedFS{FS: fstest.MapFS{
		"service/smarterr.hcl":     &fstest.MapFile{Data: []byte(`token "base" {}`)},
		"service/amp/smarterr.hcl": &fstest.MapFile{Data: []byte(`token "amp" {}`)},
	}}

	// baseDir "internal": "notinternal/service/amp/..." must match nothing.
	cfg, err := LoadConfig(context.Background(), fsys, []string{"x/y/notinternal/service/amp/file.go"}, "internal")
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	for _, tok := range cfg.Tokens {
		t.Errorf("frame under \"notinternal\" must not match any internal config; got token %q", tok.Name)
	}

	// baseDir ".": "notservice/amp/..." must not match "service/amp".
	cfg2, err := LoadConfig(context.Background(), fsys, []string{"/abs/proj/notservice/amp/file.go"}, ".")
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	for _, tok := range cfg2.Tokens {
		if tok.Name == "amp" {
			t.Errorf("frame under \"notservice\" must not match \"service/amp\"; got token %q", tok.Name)
		}
	}
}

func TestLoadConfig_ExtraConfigNotIncluded(t *testing.T) {
	fsys := &WrappedFS{FS: fstest.MapFS{
		"service/smarterr.hcl":            &fstest.MapFile{Data: []byte(`token "foo" {}`)},
		"service/cloudwatch/smarterr.hcl": &fstest.MapFile{Data: []byte(`token "bar" {}`)},
		"service/cloudtrail/smarterr.hcl": &fstest.MapFile{Data: []byte(`token "should_not_be_included" {}`)},
	}}
	relStackPaths := []string{
		"x/y/z/internal/service/cloudwatch/alarm.go",
		"x/y/z/internal/service/cloudwatch/composite_alarm.go",
	}
	baseDir := "internal"

	cfg, err := LoadConfig(context.Background(), fsys, relStackPaths, baseDir)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if len(cfg.Tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(cfg.Tokens))
	}
	names := map[string]bool{}
	for _, tok := range cfg.Tokens {
		names[tok.Name] = true
	}
	if !names["foo"] || !names["bar"] {
		t.Errorf("expected tokens 'foo' and 'bar', got: %v", names)
	}
	if names["should_not_be_included"] {
		t.Errorf("token 'should_not_be_included' should NOT be present, got: %v", names)
	}
}

func TestLoadConfig_LocalOverridesParent(t *testing.T) {
	fsys := &WrappedFS{FS: fstest.MapFS{
		"service/smarterr.hcl": &fstest.MapFile{Data: []byte(`
token "foo" {
  source = "parameter"
  parameter = "bar"
}
parameter "bar" {
  value = "parent"
}
`)},
		"service/cloudwatch/smarterr.hcl": &fstest.MapFile{Data: []byte(`
token "foo" {
  source = "parameter"
  parameter = "bar"
}
parameter "bar" {
  value = "child"
}
`)},
	}}
	relStackPaths := []string{
		"x/y/z/internal/service/cloudwatch/alarm.go",
	}
	baseDir := "internal"

	cfg, err := LoadConfig(context.Background(), fsys, relStackPaths, baseDir)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if len(cfg.Tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(cfg.Tokens))
	}
	rt := NewRuntime(context.Background(), cfg, nil, nil)
	val := cfg.Tokens[0].Resolve(context.Background(), rt)
	if val != "child" {
		t.Errorf("expected resolved token value 'child', got: %q", val)
	}
}
