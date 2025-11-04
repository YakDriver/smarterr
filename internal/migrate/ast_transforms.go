package migrate

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"slices"
	"strings"
)

// replaceSDKResourceNotFoundAST uses AST to transform SDK v2 resource not found patterns
func replaceSDKResourceNotFoundAST(content string) string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		// Fallback to original content if parsing fails
		return content
	}

	transformer := &sdkResourceNotFoundTransformer{}
	ast.Walk(transformer, file)

	if !transformer.modified {
		return content
	}

	var buf strings.Builder
	if err := format.Node(&buf, fset, file); err != nil {
		return content
	}

	result := buf.String()

	// Post-process to remove extra blank lines before closing braces
	// This fixes the formatting issue where AST transformation adds unwanted whitespace
	lines := strings.Split(result, "\n")
	var cleaned []string

	for i, line := range lines {
		// Skip blank lines that appear right before a closing brace
		if strings.TrimSpace(line) == "" && i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			if nextLine == "}" || nextLine == "})" {
				continue // Skip this blank line
			}
		}
		cleaned = append(cleaned, line)
	}

	return strings.Join(cleaned, "\n")
}

type sdkResourceNotFoundTransformer struct {
	modified bool
}

func (t *sdkResourceNotFoundTransformer) Visit(node ast.Node) ast.Visitor {
	ifStmt, ok := node.(*ast.IfStmt)
	if !ok {
		return t
	}

	// Check if this matches our pattern
	if t.isSDKResourceNotFoundPattern(ifStmt) {
		t.transformIfStatement(ifStmt)
		t.modified = true
	}

	return t
}

func (t *sdkResourceNotFoundTransformer) isSDKResourceNotFoundPattern(ifStmt *ast.IfStmt) bool {
	// Must have binary expression (&&)
	binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok || binExpr.Op != token.LAND {
		return false
	}

	// Check for both conditions in either order
	hasNotFound := t.isIntretryNotFoundCall(binExpr.X) || t.isIntretryNotFoundCall(binExpr.Y)
	hasIsNewResource := t.isNotIsNewResourceCall(binExpr.X) || t.isNotIsNewResourceCall(binExpr.Y)

	if !hasNotFound || !hasIsNewResource {
		return false
	}

	// Check body has the expected statements
	if ifStmt.Body == nil || len(ifStmt.Body.List) != 3 {
		return false
	}

	// Check for log.Printf, d.SetId(""), return diags pattern
	return t.isLogPrintfCall(ifStmt.Body.List[0]) &&
		t.isSetIdEmptyCall(ifStmt.Body.List[1]) &&
		t.isReturnDiags(ifStmt.Body.List[2])
}

func (t *sdkResourceNotFoundTransformer) isIntretryNotFoundCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	// Handle both tfresource.NotFound and intretry.NotFound
	return ok && (ident.Name == "intretry" || ident.Name == "tfresource") && sel.Sel.Name == "NotFound"
}

func (t *sdkResourceNotFoundTransformer) isNotIsNewResourceCall(expr ast.Expr) bool {
	unary, ok := expr.(*ast.UnaryExpr)
	if !ok || unary.Op != token.NOT {
		return false
	}

	call, ok := unary.X.(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == "d" && sel.Sel.Name == "IsNewResource"
}

func (t *sdkResourceNotFoundTransformer) isLogPrintfCall(stmt ast.Stmt) bool {
	exprStmt, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return false
	}

	call, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}

	return t.isLogPrintfCallExpr(call)
}

func (t *sdkResourceNotFoundTransformer) isLogPrintfCallExpr(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == "log" && sel.Sel.Name == "Printf"
}

func (t *sdkResourceNotFoundTransformer) containsDIdCall(call *ast.CallExpr) bool {
	// Walk through all arguments to find d.Id() calls
	return slices.ContainsFunc(call.Args, t.containsDIdInExpr)
}

func (t *sdkResourceNotFoundTransformer) containsDIdInExpr(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.CallExpr:
		// Check if this is d.Id()
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "d" && sel.Sel.Name == "Id" {
				return true
			}
		}
		// Recursively check arguments
		if slices.ContainsFunc(e.Args, t.containsDIdInExpr) {
			return true
		}
	case *ast.BinaryExpr:
		return t.containsDIdInExpr(e.X) || t.containsDIdInExpr(e.Y)
	case *ast.UnaryExpr:
		return t.containsDIdInExpr(e.X)
	case *ast.ParenExpr:
		return t.containsDIdInExpr(e.X)
	}
	return false
}

func (t *sdkResourceNotFoundTransformer) isSetIdEmptyCall(stmt ast.Stmt) bool {
	exprStmt, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return false
	}

	call, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != "d" || sel.Sel.Name != "SetId" {
		return false
	}

	// Check for empty string argument
	if len(call.Args) != 1 {
		return false
	}

	lit, ok := call.Args[0].(*ast.BasicLit)
	return ok && lit.Kind == token.STRING && lit.Value == `""`
}

func (t *sdkResourceNotFoundTransformer) isReturnDiags(stmt ast.Stmt) bool {
	retStmt, ok := stmt.(*ast.ReturnStmt)
	if !ok || len(retStmt.Results) != 1 {
		return false
	}

	ident, ok := retStmt.Results[0].(*ast.Ident)
	return ok && ident.Name == "diags"
}

func (t *sdkResourceNotFoundTransformer) transformIfStatement(ifStmt *ast.IfStmt) {
	// Create new condition: !d.IsNewResource() && intretry.NotFound(err)
	ifStmt.Cond = &ast.BinaryExpr{
		X: &ast.UnaryExpr{
			Op: token.NOT,
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "d"},
					Sel: &ast.Ident{Name: "IsNewResource"},
				},
			},
		},
		Op: token.LAND,
		Y: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "intretry"},
				Sel: &ast.Ident{Name: "NotFound"},
			},
			Args: []ast.Expr{&ast.Ident{Name: "err"}},
		},
	}

	// Preserve the original block structure to avoid extra whitespace
	originalBody := ifStmt.Body

	// Extract ID from the original log.Printf call if present
	var idArgs []ast.Expr
	if len(originalBody.List) > 0 {
		if exprStmt, ok := originalBody.List[0].(*ast.ExprStmt); ok {
			if call, ok := exprStmt.X.(*ast.CallExpr); ok {
				if t.isLogPrintfCallExpr(call) && t.containsDIdCall(call) {
					// Add smerr.ID and d.Id() to preserve the resource ID
					idArgs = []ast.Expr{
						&ast.SelectorExpr{
							X:   &ast.Ident{Name: "smerr"},
							Sel: &ast.Ident{Name: "ID"},
						},
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   &ast.Ident{Name: "d"},
								Sel: &ast.Ident{Name: "Id"},
							},
						},
					}
				}
			}
		}
	}

	// Create base arguments for smerr.AppendOne
	baseArgs := []ast.Expr{
		&ast.Ident{Name: "ctx"},
		&ast.Ident{Name: "diags"},
		&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "sdkdiag"},
				Sel: &ast.Ident{Name: "NewResourceNotFoundWarningDiagnostic"},
			},
			Args: []ast.Expr{&ast.Ident{Name: "err"}},
		},
	}

	// Append ID arguments if found
	allArgs := append(baseArgs, idArgs...)

	// Create new body statements
	newStmts := []ast.Stmt{
		// smerr.AppendOne(ctx, diags, sdkdiag.NewResourceNotFoundWarningDiagnostic(err)[, smerr.ID, d.Id()])
		&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "smerr"},
					Sel: &ast.Ident{Name: "AppendOne"},
				},
				Args: allArgs,
			},
		},
		// d.SetId("")
		&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "d"},
					Sel: &ast.Ident{Name: "SetId"},
				},
				Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `""`}},
			},
		},
		// return diags
		&ast.ReturnStmt{
			Results: []ast.Expr{&ast.Ident{Name: "diags"}},
		},
	}

	// Update the body while preserving the original block structure
	originalBody.List = newStmts
}
