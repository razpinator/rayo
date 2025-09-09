package sem

import (
    "testing"
    "functure/internal/ast"
    "functure/internal/diag"
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
