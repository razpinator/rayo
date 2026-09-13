package gen

import (
	"go/format"
	"testing"

	"rayo/internal/ast"
)

func TestEmitGenericFunc(t *testing.T) {
	fn := &ast.FuncDef{
		Name:       "identity",
		TypeParams: []string{"T"},
		Params:     []*ast.Param{{Name: "x", Type: &ast.Named{Name: "T"}}},
		RetType:    &ast.Named{Name: "T"},
		Body:       []ast.Stmt{&ast.ReturnStmt{Value: &ast.Name{Ident: "x"}}},
	}
	ctx := NewGenContext("main")
	EmitStmt(fn, ctx)
	code := ctx.Code.String()
	if !contains(code, "func identity[T any](x T) T {") {
		t.Errorf("generic signature not emitted: %s", code)
	}
}

func TestEmitTypedParamsMapToGo(t *testing.T) {
	fn := &ast.FuncDef{
		Name: "add",
		Params: []*ast.Param{
			{Name: "a", Type: &ast.Named{Name: "int"}},
			{Name: "b", Type: &ast.Named{Name: "float"}},
		},
		RetType: &ast.Named{Name: "str"},
		Body:    []ast.Stmt{&ast.ReturnStmt{Value: &ast.Literal{Value: "x"}}},
	}
	ctx := NewGenContext("main")
	EmitStmt(fn, ctx)
	code := ctx.Code.String()
	if !contains(code, "func add(a int64, b float64) string {") {
		t.Errorf("typed params not mapped to Go: %s", code)
	}
}

func TestGoTypeMapping(t *testing.T) {
	tp := map[string]bool{"T": true}
	cases := []struct {
		in   ast.Type
		want string
	}{
		{&ast.Named{Name: "int"}, "int64"},
		{&ast.Named{Name: "float"}, "float64"},
		{&ast.Named{Name: "str"}, "string"},
		{&ast.Named{Name: "bool"}, "bool"},
		{&ast.Named{Name: "T"}, "T"},
		{&ast.Named{Name: "list", Args: []ast.Type{&ast.Named{Name: "int"}}}, "[]int64"},
		{&ast.Named{Name: "dict", Args: []ast.Type{&ast.Named{Name: "str"}, &ast.Named{Name: "int"}}}, "map[string]int64"},
		{&ast.Optional{Elem: &ast.Named{Name: "int"}}, "*int64"},
		{&ast.Named{Name: "Widget"}, "any"}, // unknown named type stays dynamic
		{&ast.Any{}, "any"},
		{nil, "any"},
	}
	for _, c := range cases {
		if got := goType(c.in, tp); got != c.want {
			t.Errorf("goType(%#v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestEmitGenericFuncIsValidGo(t *testing.T) {
	// A whole module with a generic identity function must format as valid Go.
	mod := &ast.Module{
		Body: []ast.Stmt{
			&ast.FuncDef{
				Name:       "identity",
				TypeParams: []string{"T"},
				Params:     []*ast.Param{{Name: "x", Type: &ast.Named{Name: "T"}}},
				RetType:    &ast.Named{Name: "T"},
				Body:       []ast.Stmt{&ast.ReturnStmt{Value: &ast.Name{Ident: "x"}}},
			},
		},
	}
	code := EmitModule(mod, NewGenContext("main"))
	if _, err := format.Source([]byte(code)); err != nil {
		t.Fatalf("generated generic code is not valid Go: %v\n%s", err, code)
	}
}
