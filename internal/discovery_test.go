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

	cfg, err := LoadConfig(fsys, relStackPaths, baseDir)
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

	cfg, err := LoadConfig(fsys, relStackPaths, baseDir)
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

	cfg, err := LoadConfig(fsys, relStackPaths, baseDir)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if len(cfg.Tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(cfg.Tokens))
	}
	rt := NewRuntime(cfg, nil, nil)
	val := cfg.Tokens[0].Resolve(context.Background(), rt)
	if val != "child" {
		t.Errorf("expected resolved token value 'child', got: %q", val)
	}
}
