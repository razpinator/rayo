package gen

import (
	"go/format"
	"testing"

	"rayo/internal/ast"
)

// pointClass builds a small class AST used across the codegen tests.
func pointClass() *ast.ClassDef {
	return &ast.ClassDef{
		Name: "Point",
		Fields: []*ast.Field{
			{Name: "x", Type: &ast.Named{Name: "int"}},
			{Name: "y", Type: &ast.Named{Name: "int"}},
		},
		Methods: []*ast.FuncDef{
			{
				Name: "__init__",
				Params: []*ast.Param{
					{Name: "self"},
					{Name: "x", Type: &ast.Named{Name: "int"}},
					{Name: "y", Type: &ast.Named{Name: "int"}},
				},
				Body: []ast.Stmt{
					&ast.AssignStmt{Target: &ast.Attr{Target: &ast.Name{Ident: "self"}, Attr: "x"}, Value: &ast.Name{Ident: "x"}},
					&ast.AssignStmt{Target: &ast.Attr{Target: &ast.Name{Ident: "self"}, Attr: "y"}, Value: &ast.Name{Ident: "y"}},
				},
			},
			{
				Name:    "sum",
				Params:  []*ast.Param{{Name: "self"}},
				RetType: &ast.Named{Name: "int"},
				Body: []ast.Stmt{
					&ast.ReturnStmt{Value: &ast.BinaryOp{Op: "+",
						Left:  &ast.Attr{Target: &ast.Name{Ident: "self"}, Attr: "x"},
						Right: &ast.Attr{Target: &ast.Name{Ident: "self"}, Attr: "y"}}},
				},
			},
		},
	}
}

func TestEmitClassStructAndConstructor(t *testing.T) {
	ctx := NewGenContext("main")
	EmitStmt(pointClass(), ctx)
	code := ctx.Code.String()
	if !contains(code, "type Point struct {") {
		t.Errorf("struct not emitted: %s", code)
	}
	if !contains(code, "x int64") || !contains(code, "y int64") {
		t.Errorf("typed fields not emitted: %s", code)
	}
	if !contains(code, "func NewPoint(x int64, y int64) *Point {") {
		t.Errorf("constructor not emitted from __init__: %s", code)
	}
	if !contains(code, "self := &Point{}") || !contains(code, "return self") {
		t.Errorf("constructor body wrong: %s", code)
	}
	if !contains(code, "func (self *Point) sum() int64 {") {
		t.Errorf("method receiver not emitted: %s", code)
	}
}

func TestEmitClassProperty(t *testing.T) {
	cls := &ast.ClassDef{
		Name:   "Box",
		Fields: []*ast.Field{{Name: "w", Type: &ast.Named{Name: "int"}}},
		Properties: []*ast.Property{
			{
				Name: "area",
				Type: &ast.Named{Name: "int"},
				Get:  []ast.Stmt{&ast.ReturnStmt{Value: &ast.Attr{Target: &ast.Name{Ident: "self"}, Attr: "w"}}},
				Set: []ast.Stmt{&ast.AssignStmt{
					Target: &ast.Attr{Target: &ast.Name{Ident: "self"}, Attr: "w"},
					Value:  &ast.Name{Ident: "value"}}},
				SetParam: "value",
			},
		},
	}
	ctx := NewGenContext("main")
	EmitStmt(cls, ctx)
	code := ctx.Code.String()
	if !contains(code, "func (self *Box) area() int64 {") {
		t.Errorf("property getter not emitted: %s", code)
	}
	if !contains(code, "func (self *Box) SetArea(value int64) {") {
		t.Errorf("property setter not emitted: %s", code)
	}
}

func TestEmitClassConstructorCallRewrite(t *testing.T) {
	// A module where a class is constructed by name should rewrite to NewPoint.
	mod := &ast.Module{
		Body: []ast.Stmt{
			pointClass(),
			&ast.AssignStmt{Target: &ast.Name{Ident: "p"}, Value: &ast.Call{
				Func: &ast.Name{Ident: "Point"},
				Args: []ast.Expr{&ast.Literal{Value: 1}, &ast.Literal{Value: 2}},
			}},
		},
	}
	code := EmitModule(mod, NewGenContext("main"))
	if !contains(code, "NewPoint(1, 2)") {
		t.Errorf("constructor call not rewritten to NewPoint: %s", code)
	}
}

func TestEmitClassIsValidGo(t *testing.T) {
	mod := &ast.Module{Body: []ast.Stmt{pointClass()}}
	code := EmitModule(mod, NewGenContext("main"))
	if _, err := format.Source([]byte(code)); err != nil {
		t.Fatalf("generated class code is not valid Go: %v\n%s", err, code)
	}
}

func TestEmitClassBaseEmbedding(t *testing.T) {
	cls := &ast.ClassDef{Name: "Dog", Base: "Animal", Fields: []*ast.Field{{Name: "name", Type: &ast.Named{Name: "str"}}}}
	ctx := NewGenContext("main")
	EmitStmt(cls, ctx)
	code := ctx.Code.String()
	if !contains(code, "type Dog struct {") || !contains(code, "Animal\n") {
		t.Errorf("base class not embedded: %s", code)
	}
}
