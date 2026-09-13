package opt

import (
	"testing"

	"rayo/internal/ast"
)

func lit(v any) *ast.Literal { return &ast.Literal{Value: v} }

func TestFoldBinaryArithmetic(t *testing.T) {
	cases := []struct {
		op   string
		a, b any
		want any
	}{
		{"+", 2, 3, 5},
		{"-", 10, 4, 6},
		{"*", 6, 7, 42},
		{"/", 8, 2, 4},
		{"%", 10, 3, 1},
		{"+", 1.5, 2.5, 4.0},
		{"*", 2, 2.5, 5.0},
		{"+", "foo", "bar", "foobar"},
		{"==", 1, 1, true},
		{"<", 1, 2, true},
		{">=", 2, 3, false},
		{"and", true, false, false},
		{"or", false, true, true},
	}
	for _, c := range cases {
		got := foldExpr(&ast.BinaryOp{Op: c.op, Left: lit(c.a), Right: lit(c.b)})
		l, ok := got.(*ast.Literal)
		if !ok {
			t.Errorf("%v %s %v: not folded to a literal (got %T)", c.a, c.op, c.b, got)
			continue
		}
		if l.Value != c.want {
			t.Errorf("%v %s %v: got %v, want %v", c.a, c.op, c.b, l.Value, c.want)
		}
	}
}

func TestFoldUnary(t *testing.T) {
	if l, ok := foldExpr(&ast.UnaryOp{Op: "-", Right: lit(5)}).(*ast.Literal); !ok || l.Value != -5 {
		t.Errorf("unary minus not folded")
	}
	if l, ok := foldExpr(&ast.UnaryOp{Op: "not", Right: lit(false)}).(*ast.Literal); !ok || l.Value != true {
		t.Errorf("not false should fold to true")
	}
}

func TestFoldNested(t *testing.T) {
	// (2 + 3) * 4 -> 20
	expr := &ast.BinaryOp{
		Op:   "*",
		Left: &ast.BinaryOp{Op: "+", Left: lit(2), Right: lit(3)},
		Right: lit(4),
	}
	got := foldExpr(expr)
	l, ok := got.(*ast.Literal)
	if !ok || l.Value != 20 {
		t.Errorf("nested fold failed: got %#v", got)
	}
}

func TestFoldLeavesNonConstant(t *testing.T) {
	// x + 1 must not fold (x is not a literal), but 1 + 2 inside should.
	expr := &ast.BinaryOp{
		Op:    "+",
		Left:  &ast.Name{Ident: "x"},
		Right: &ast.BinaryOp{Op: "+", Left: lit(1), Right: lit(2)},
	}
	got := foldExpr(expr)
	bin, ok := got.(*ast.BinaryOp)
	if !ok {
		t.Fatalf("expected BinaryOp to remain, got %T", got)
	}
	r, ok := bin.Right.(*ast.Literal)
	if !ok || r.Value != 3 {
		t.Errorf("inner constant subtree not folded: %#v", bin.Right)
	}
}

func TestFoldDivByZeroLeftAlone(t *testing.T) {
	got := foldExpr(&ast.BinaryOp{Op: "/", Left: lit(1), Right: lit(0)})
	if _, ok := got.(*ast.Literal); ok {
		t.Errorf("division by zero must not be folded")
	}
}

func TestFoldModuleWalksBodies(t *testing.T) {
	mod := &ast.Module{
		Body: []ast.Stmt{
			&ast.FuncDef{Name: "f", Body: []ast.Stmt{
				&ast.ReturnStmt{Value: &ast.BinaryOp{Op: "*", Left: lit(6), Right: lit(7)}},
			}},
		},
	}
	FoldModule(mod)
	fn := mod.Body[0].(*ast.FuncDef)
	ret := fn.Body[0].(*ast.ReturnStmt)
	l, ok := ret.Value.(*ast.Literal)
	if !ok || l.Value != 42 {
		t.Errorf("FoldModule did not fold function body: %#v", ret.Value)
	}
}
