package parse

import (
	"testing"

	"rayo/internal/ast"
)

func TestParseClassFieldsAndMethods(t *testing.T) {
	src := `class Point {
    x: int
    y: int
    def __init__(self, x: int, y: int) { self.x = x self.y = y }
    def sum(self) -> int { return self.x + self.y }
}`
	mod := mustParse(t, src)
	cls, ok := mod.Body[0].(*ast.ClassDef)
	if !ok {
		t.Fatalf("expected ClassDef, got %T", mod.Body[0])
	}
	if cls.Name != "Point" {
		t.Errorf("class name: got %q", cls.Name)
	}
	if len(cls.Fields) != 2 || cls.Fields[0].Name != "x" || cls.Fields[1].Name != "y" {
		t.Errorf("fields not parsed: %+v", cls.Fields)
	}
	if n, ok := cls.Fields[0].Type.(*ast.Named); !ok || n.Name != "int" {
		t.Errorf("field type not int: %#v", cls.Fields[0].Type)
	}
	if len(cls.Methods) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(cls.Methods))
	}
	if cls.Methods[0].Name != "__init__" || cls.Methods[1].Name != "sum" {
		t.Errorf("method names wrong: %s, %s", cls.Methods[0].Name, cls.Methods[1].Name)
	}
	// __init__ carries self + two params.
	if len(cls.Methods[0].Params) != 3 || cls.Methods[0].Params[0].Name != "self" {
		t.Errorf("__init__ params wrong: %+v", cls.Methods[0].Params)
	}
}

func TestParseClassBase(t *testing.T) {
	mod := mustParse(t, `class Dog(Animal) { def speak(self) { return "woof" } }`)
	cls := mod.Body[0].(*ast.ClassDef)
	if cls.Base != "Animal" {
		t.Errorf("base class not parsed: %q", cls.Base)
	}
}

func TestParseClassProperty(t *testing.T) {
	src := `class Box {
    w: int
    property area {
        get { return self.w * self.w }
        set(value) { self.w = value }
    }
}`
	mod := mustParse(t, src)
	cls := mod.Body[0].(*ast.ClassDef)
	if len(cls.Properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(cls.Properties))
	}
	prop := cls.Properties[0]
	if prop.Name != "area" {
		t.Errorf("property name: %q", prop.Name)
	}
	if len(prop.Get) != 1 {
		t.Errorf("getter body not parsed: %+v", prop.Get)
	}
	if len(prop.Set) != 1 {
		t.Errorf("setter body not parsed: %+v", prop.Set)
	}
	if prop.SetParam != "value" {
		t.Errorf("setter param: %q", prop.SetParam)
	}
}
