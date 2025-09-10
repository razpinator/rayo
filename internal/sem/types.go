package sem

import "rayo/internal/ast"

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
        switch e.Value.(type) {
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
        // Example: lookup in symbol table for null safety
        // (symbol table logic would go here)
        return &AnyType{}
    case *ast.Attr:
        // Disambiguate obj.attr vs obj["attr"]
        // If Target is known struct, return field type; else dynamic
        if bt, ok := InferType(e.Target).(*BasicType); ok && bt.Name == "struct" {
            return &AnyType{} // Would be field type in real impl
        }
        return &AnyType{}
    case *ast.Index:
        // If Target is dict, return value type; else dynamic
        if bt, ok := InferType(e.Target).(*BasicType); ok && bt.Name == "dict" {
            return &AnyType{} // Would be value type in real impl
        }
        return &AnyType{}
    default:
        return &AnyType{}
    }
}
