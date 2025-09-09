package ast

import (
    "testing"
    "functure/internal/diag"
)

func TestASTRoundTrip(t *testing.T) {
    span := diag.Span{}
    mod := &Module{
        Name: "test",
        Imports: []*Import{{Path: "core", span: span}},
        Body: []Stmt{
            &VarStmt{Name: "x", Value: NewLiteral(42, span), span: span},
            &IfStmt{
                Cond: NewName("x", span),
                Then: []Stmt{&ReturnStmt{Value: NewName("x", span), span: span}},
                Elifs: []*Elif{},
                Else: []Stmt{},
                span: span,
            },
        },
        span: span,
    }
    var nodes []Node
    v := &collectVisitor{nodes: &nodes}
    Walk(v, mod)
    if len(nodes) < 5 {
        t.Errorf("expected AST to have at least 5 nodes, got %d", len(nodes))
    }
}

type collectVisitor struct {
    nodes *[]Node
}

func (v *collectVisitor) Visit(n Node) bool {
    *v.nodes = append(*v.nodes, n)
    return true
}
