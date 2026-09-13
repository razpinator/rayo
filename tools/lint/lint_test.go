package lint

import (
	"testing"

	"rayo/internal/ast"
	"rayo/internal/parse"
)

// hasRule reports whether any finding carries the given rule ID.
func hasRule(results []LintResult, id string) bool {
	for _, r := range results {
		if r.RuleID == id {
			return true
		}
	}
	return false
}

func lintSource(t *testing.T, src string) []LintResult {
	t.Helper()
	p := parse.NewParser(src)
	mod := p.ParseModule()
	if mod == nil {
		t.Fatalf("parse returned nil module for %q", src)
	}
	return LintModule(mod)
}

// TestRY001_UnusedBinding covers the unused-variable rule via a hand-built AST
// (parser support for arbitrary bodies is partial, so we construct directly).
func TestRY001_UnusedBinding(t *testing.T) {
	mod := &ast.Module{
		Name: "test",
		Body: []ast.Stmt{
			&ast.VarStmt{Name: "x", Value: &ast.Literal{Value: 1}},
		},
	}
	results := LintModule(mod)
	if !hasRule(results, RuleUnusedBinding) {
		t.Errorf("expected %s (unused binding), got %v", RuleUnusedBinding, results)
	}

	// A used binding should not fire RY001.
	mod2 := &ast.Module{
		Name: "test",
		Body: []ast.Stmt{
			&ast.VarStmt{Name: "y", Value: &ast.Literal{Value: 1}},
			&ast.ExprStmt{Expr: &ast.Name{Ident: "y"}},
		},
	}
	if hasRule(LintModule(mod2), RuleUnusedBinding) {
		t.Errorf("did not expect %s for a used binding", RuleUnusedBinding)
	}
}

// TestRY002_UnusedImport checks that imports whose alias is never referenced
// are flagged, and referenced imports are not.
func TestRY002_UnusedImport(t *testing.T) {
	unused := &ast.Module{
		Name:    "test",
		Imports: []*ast.Import{{Path: "rayo/stdlib/data"}},
		Body:    []ast.Stmt{},
	}
	if !hasRule(LintModule(unused), RuleUnusedImport) {
		t.Errorf("expected %s for unreferenced import", RuleUnusedImport)
	}

	used := &ast.Module{
		Name:    "test",
		Imports: []*ast.Import{{Path: "rayo/stdlib/data"}},
		Body: []ast.Stmt{
			&ast.ExprStmt{Expr: &ast.Call{
				Func: &ast.Attr{Target: &ast.Name{Ident: "data"}, Attr: "map"},
			}},
		},
	}
	if hasRule(LintModule(used), RuleUnusedImport) {
		t.Errorf("did not expect %s when the import alias is used", RuleUnusedImport)
	}
}

// TestRY003_SuspiciousAttr fires when a dict literal is accessed via .attr.
func TestRY003_SuspiciousAttr(t *testing.T) {
	mod := &ast.Module{
		Name: "test",
		Body: []ast.Stmt{
			&ast.ExprStmt{Expr: &ast.Attr{
				Target: &ast.DictLit{Keys: []ast.Expr{}, Vals: []ast.Expr{}},
				Attr:   "foo",
			}},
		},
	}
	if !hasRule(LintModule(mod), RuleSuspiciousAttr) {
		t.Errorf("expected %s for attr access on a dict literal", RuleSuspiciousAttr)
	}

	// Attribute access on a plain (any) name should NOT fire RY003 — that would
	// be too noisy.
	quiet := &ast.Module{
		Name: "test",
		Body: []ast.Stmt{
			&ast.VarStmt{Name: "obj", Value: &ast.Literal{Value: 1}},
			&ast.ExprStmt{Expr: &ast.Attr{Target: &ast.Name{Ident: "obj"}, Attr: "foo"}},
		},
	}
	if hasRule(LintModule(quiet), RuleSuspiciousAttr) {
		t.Errorf("did not expect %s for attr access on a non-dict value", RuleSuspiciousAttr)
	}
}

// TestRY004_UnsafeOptionalDeref fires when a None-typed value is dereferenced.
func TestRY004_UnsafeOptionalDeref(t *testing.T) {
	mod := &ast.Module{
		Name: "test",
		Body: []ast.Stmt{
			&ast.ExprStmt{Expr: &ast.Attr{
				Target: &ast.Literal{Value: nil}, // None -> optional
				Attr:   "foo",
			}},
		},
	}
	if !hasRule(LintModule(mod), RuleUnsafeOptionalRef) {
		t.Errorf("expected %s for dereference of optional value", RuleUnsafeOptionalRef)
	}
}

// TestRY005_HighComplexity fires when a function exceeds the complexity
// threshold, and stays quiet for simple functions.
func TestRY005_HighComplexity(t *testing.T) {
	// Build a function with many branches to exceed the default threshold (10).
	var body []ast.Stmt
	for i := 0; i < 12; i++ {
		body = append(body, &ast.IfStmt{
			Cond: &ast.Literal{Value: true},
			Then: []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 1}}},
		})
	}
	mod := &ast.Module{
		Name: "test",
		Body: []ast.Stmt{&ast.FuncDef{Name: "big", Body: body}},
	}
	results := LintModule(mod)
	if !hasRule(results, RuleHighComplexity) {
		t.Errorf("expected %s for a highly branching function, got %v", RuleHighComplexity, results)
	}

	// A trivial function must not fire RY005.
	simple := &ast.Module{
		Name: "test",
		Body: []ast.Stmt{&ast.FuncDef{
			Name: "small",
			Body: []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: 1}}},
		}},
	}
	if hasRule(LintModule(simple), RuleHighComplexity) {
		t.Errorf("did not expect %s for a simple function", RuleHighComplexity)
	}
}

// TestLintModule_Deterministic ensures results are sorted stably by position.
func TestLintModule_Deterministic(t *testing.T) {
	src := "import 'rayo/stdlib/data'\nvar x = 42"
	first := lintSource(t, src)
	second := lintSource(t, src)
	if len(first) != len(second) {
		t.Fatalf("nondeterministic result count: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].RuleID != second[i].RuleID {
			t.Errorf("nondeterministic ordering at %d: %s vs %s", i, first[i].RuleID, second[i].RuleID)
		}
	}
}
