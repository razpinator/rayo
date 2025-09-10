package sem

import (
    "testing"
    "rayo/internal/ast"
    "rayo/internal/diag"
)

type testReporter struct {
    errors []string
}

func (r *testReporter) Report(span diag.Span, msg string) {
    r.errors = append(r.errors, msg)
}

func TestUnusedVarWarning(t *testing.T) {
    mod := &ast.Module{
        Name: "test",
        Imports: []*ast.Import{},
        Body: []ast.Stmt{
            &ast.VarStmt{Name: "x", Value: &ast.Literal{Value: 1}},
        },
    }
    rep := &testReporter{}
    CheckModule(mod, rep)
    found := false
    for _, err := range rep.errors {
        if err == "unused variable: x" {
            found = true
        }
    }
    if !found {
        t.Errorf("expected unused variable warning")
    }
}

func TestNullSafetyDiagnostics(t *testing.T) {
    // Unsafe dereference of optional value
    expr := &ast.Attr{Target: &ast.Literal{Value: nil}}
    stmt := &ast.ExprStmt{Expr: expr}
    mod := &ast.Module{Body: []ast.Stmt{stmt}}
    rep := &testReporter{}
    CheckModule(mod, rep)
    found := false
    for _, err := range rep.errors {
        if err == "unsafe dereference of optional value" {
            found = true
        }
    }
    if !found {
        t.Errorf("expected null safety diagnostic")
    }
}

func TestIndexNullSafetyDiagnostics(t *testing.T) {
    expr := &ast.Index{Target: &ast.Literal{Value: nil}, Index: &ast.Literal{Value: 0}}
    stmt := &ast.ExprStmt{Expr: expr}
    mod := &ast.Module{Body: []ast.Stmt{stmt}}
    rep := &testReporter{}
    CheckModule(mod, rep)
    found := false
    for _, err := range rep.errors {
        if err == "unsafe index of optional value" {
            found = true
        }
    }
    if !found {
        t.Errorf("expected index null safety diagnostic")
    }
}
