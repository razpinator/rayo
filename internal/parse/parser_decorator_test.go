package parse

import (
	"testing"

	"rayo/internal/ast"
)

func TestParseSingleDecorator(t *testing.T) {
	mod := mustParse(t, "@log\ndef f(x) { return x }")
	fn, ok := mod.Body[0].(*ast.FuncDef)
	if !ok {
		t.Fatalf("expected FuncDef, got %T", mod.Body[0])
	}
	if len(fn.Decorators) != 1 {
		t.Fatalf("expected 1 decorator, got %d", len(fn.Decorators))
	}
	if n, ok := fn.Decorators[0].(*ast.Name); !ok || n.Ident != "log" {
		t.Errorf("decorator should be Name 'log': %#v", fn.Decorators[0])
	}
}

func TestParseMultipleDecoratorsOrder(t *testing.T) {
	mod := mustParse(t, "@a\n@b\ndef f(x) { return x }")
	fn := mod.Body[0].(*ast.FuncDef)
	if len(fn.Decorators) != 2 {
		t.Fatalf("expected 2 decorators, got %d", len(fn.Decorators))
	}
	// Stored outermost-first, as written top-to-bottom.
	if n := fn.Decorators[0].(*ast.Name); n.Ident != "a" {
		t.Errorf("first decorator should be 'a', got %q", n.Ident)
	}
	if n := fn.Decorators[1].(*ast.Name); n.Ident != "b" {
		t.Errorf("second decorator should be 'b', got %q", n.Ident)
	}
}

func TestParseCallDecorator(t *testing.T) {
	mod := mustParse(t, "@retry(3)\ndef f(x) { return x }")
	fn := mod.Body[0].(*ast.FuncDef)
	if len(fn.Decorators) != 1 {
		t.Fatalf("expected 1 decorator, got %d", len(fn.Decorators))
	}
	call, ok := fn.Decorators[0].(*ast.Call)
	if !ok {
		t.Fatalf("decorator should be a Call, got %T", fn.Decorators[0])
	}
	if n, ok := call.Func.(*ast.Name); !ok || n.Ident != "retry" {
		t.Errorf("decorator callee should be 'retry': %#v", call.Func)
	}
	if len(call.Args) != 1 {
		t.Errorf("decorator call should have 1 arg, got %d", len(call.Args))
	}
}

func TestParseUndecoratedFuncHasNoDecorators(t *testing.T) {
	mod := mustParse(t, "def f(x) { return x }")
	fn := mod.Body[0].(*ast.FuncDef)
	if len(fn.Decorators) != 0 {
		t.Errorf("undecorated function should have no decorators, got %d", len(fn.Decorators))
	}
}
