package lint

import (
    "testing"
    "rayo/internal/ast"
)

func TestLintModule(t *testing.T) {
    mod := &ast.Module{
        Name: "test",
        Imports: []*ast.Import{},
        Body: []ast.Stmt{
            &ast.VarStmt{Name: "x", Value: &ast.Literal{Value: 1}},
            &ast.ExprStmt{Expr: &ast.Attr{Target: &ast.Literal{Value: nil}, Attr: "foo"}},
        },
    }
    results := LintModule(mod)
    foundUnused := false
    foundNull := false
    foundSuspicious := false
    for _, r := range results {
        if r.Msg == "unused variable: x" {
            foundUnused = true
        }
        if r.Msg == "possible unsafe dereference of optional value" {
            foundNull = true
        }
        if r.Msg == "suspicious attribute access on dict-like object" {
            foundSuspicious = true
        }
    }
    if !foundUnused {
        t.Errorf("Expected unused variable warning")
    }
    if !foundNull {
        t.Errorf("Expected null safety hint")
    }
    if !foundSuspicious {
        t.Errorf("Expected suspicious attr vs key warning")
    }
}
