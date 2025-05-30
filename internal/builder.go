package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// Builder helps construct a smarterr append operation in a fluent way.
type Builder struct {
	ctx    context.Context
	err    error
	diags  diag.Diagnostics
	params map[string]any
}

// With initializes the builder with a context.
func With(ctx context.Context) *Builder {
	return &Builder{
		ctx:    ctx,
		params: make(map[string]any),
	}
}

// Error sets the main error to be used in the append.
func (b *Builder) Error(err error) *Builder {
	b.err = err
	return b
}

// Diags allows adding diagnostics in multiple formats.
func (b *Builder) Diags(v any) *Builder {
	switch d := v.(type) {
	case diag.Diagnostics:
		b.diags = append(b.diags, d...)
	case []error:
		//for _, e := range d {
		//b.diags = append(b.diags, diag.FromErr(e)...)
		//}
	case error:
		//b.diags = append(b.diags, diag.FromErr(d)...)
	default:
		// Optionally: silently ignore or log? For now, just do nothing.
	}
	return b
}

// Param sets a single named parameter for token resolution.
func (b *Builder) Param(name string, value any) *Builder {
	b.params[name] = value
	return b
}

// Append performs the actual smarterr append operation.
// You would plug in your internal Resolve + Config logic here.
func (b *Builder) Append() diag.Diagnostics {
	// Example stub logic – replace with smarter token resolving, formatting, etc.
	if b.err != nil {
		formatted := fmt.Sprintf("error: %v", b.err)

		// Add any useful params to the message (for demo/debug purposes)
		if len(b.params) > 0 {
			formatted += " | params: "
			for k, v := range b.params {
				formatted += fmt.Sprintf("%s=%v ", k, v)
			}
		}

		b.diags = append(b.diags, diag.NewErrorDiagnostic("Smarterr", formatted))
	}

	return b.diags
}
