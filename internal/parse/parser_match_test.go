package parse

import (
	"testing"

	"rayo/internal/ast"
)

func TestParseMatchLiteralCases(t *testing.T) {
	src := `def f(n) {
    match n {
        case 0 { return "zero" }
        case 1 { return "one" }
    }
}`
	mod := mustParse(t, src)
	fn := mod.Body[0].(*ast.FuncDef)
	m, ok := fn.Body[0].(*ast.MatchStmt)
	if !ok {
		t.Fatalf("expected MatchStmt, got %T", fn.Body[0])
	}
	if _, ok := m.Subject.(*ast.Name); !ok {
		t.Errorf("subject should be a Name, got %T", m.Subject)
	}
	if len(m.Cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(m.Cases))
	}
	if lit, ok := m.Cases[0].Pattern.(*ast.Literal); !ok || lit.Value != 0 {
		t.Errorf("first case pattern should be literal 0: %#v", m.Cases[0].Pattern)
	}
	if m.Cases[0].Binding != "" || m.Cases[0].IsWildcard {
		t.Errorf("literal case should not be a capture/wildcard")
	}
}

func TestParseMatchCaptureAndWildcard(t *testing.T) {
	src := `def f(n) {
    match n {
        case 1 { return "one" }
        case other { return other }
    }
}`
	mod := mustParse(t, src)
	fn := mod.Body[0].(*ast.FuncDef)
	m := fn.Body[0].(*ast.MatchStmt)
	if m.Cases[1].Binding != "other" {
		t.Errorf("expected capture binding 'other', got %q (wildcard=%v pattern=%#v)",
			m.Cases[1].Binding, m.Cases[1].IsWildcard, m.Cases[1].Pattern)
	}

	src2 := `def g(n) {
    match n {
        case 1 { return "one" }
        case _ { return "default" }
    }
}`
	mod2 := mustParse(t, src2)
	fn2 := mod2.Body[0].(*ast.FuncDef)
	m2 := fn2.Body[0].(*ast.MatchStmt)
	if !m2.Cases[1].IsWildcard {
		t.Errorf("expected wildcard case, got binding=%q pattern=%#v", m2.Cases[1].Binding, m2.Cases[1].Pattern)
	}
}

func TestParseMatchExpressionPattern(t *testing.T) {
	// A pattern that is an expression starting with an identifier (not a bare
	// capture) must be parsed as an expression, not a binding.
	src := `def f(n) {
    match n {
        case a + 1 { return "x" }
    }
}`
	mod := mustParse(t, src)
	fn := mod.Body[0].(*ast.FuncDef)
	m := fn.Body[0].(*ast.MatchStmt)
	if m.Cases[0].Binding != "" {
		t.Errorf("expression pattern must not be treated as a capture binding: %q", m.Cases[0].Binding)
	}
	if _, ok := m.Cases[0].Pattern.(*ast.BinaryOp); !ok {
		t.Errorf("expected BinaryOp pattern, got %#v", m.Cases[0].Pattern)
	}
}
