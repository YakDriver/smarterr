package migrate

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
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

	return buf.String()
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
	return ok && ident.Name == "intretry" && sel.Sel.Name == "NotFound"
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

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == "log" && sel.Sel.Name == "Printf"
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

	// Create new body statements
	ifStmt.Body.List = []ast.Stmt{
		// smerr.AppendOne(ctx, diags, sdkdiag.NewResourceNotFoundWarningDiagnostic(err))
		&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "smerr"},
					Sel: &ast.Ident{Name: "AppendOne"},
				},
				Args: []ast.Expr{
					&ast.Ident{Name: "ctx"},
					&ast.Ident{Name: "diags"},
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   &ast.Ident{Name: "sdkdiag"},
							Sel: &ast.Ident{Name: "NewResourceNotFoundWarningDiagnostic"},
						},
						Args: []ast.Expr{&ast.Ident{Name: "err"}},
					},
				},
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
}
