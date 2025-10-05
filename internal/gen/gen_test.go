package gen

import (
	"rayo/internal/ast"
	"strings"
	"testing"
)

func TestEmitModule(t *testing.T) {
	mod := &ast.Module{
		Name:    "main",
		Imports: []*ast.Import{{Path: "fmt"}},
		Body: []ast.Stmt{
			&ast.VarStmt{Name: "x", Value: &ast.Literal{Value: 42}},
			&ast.ReturnStmt{Value: &ast.Name{Ident: "x"}},
		},
	}
	ctx := NewGenContext("main")
	code := EmitModule(mod, ctx)
	if !contains(code, "var x = 42") || !contains(code, "return x") {
		t.Errorf("codegen failed: %s", code)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
