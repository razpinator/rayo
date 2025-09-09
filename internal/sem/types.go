package sem

import "functure/internal/ast"

// Type system for semantic analysis

type Type interface{}

type BasicType struct {
    Name string // "int", "str", "bool", etc.
}

type OptionalType struct {
    Elem Type
}

type AnyType struct{}

// InferType infers the type of an AST expression.
func InferType(expr ast.Expr) Type {
    switch e := expr.(type) {
    case *ast.Literal:
        switch v := e.Value.(type) {
        case int:
            return &BasicType{Name: "int"}
        case string:
            return &BasicType{Name: "str"}
        case bool:
            return &BasicType{Name: "bool"}
        case nil:
            return &OptionalType{Elem: &AnyType{}}
        default:
            return &AnyType{}
        }
    case *ast.Name:
        return &AnyType{} // Needs symbol table lookup
    default:
        return &AnyType{}
    }
}
