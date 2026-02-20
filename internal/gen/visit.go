package gen

import (
"rayo/internal/ast"
)

func ContainsPrint(n ast.Node) bool {
    v := &printFinder{}
    ast.Walk(v, n)
    return v.found
}

type printFinder struct {
    found bool
}

func (v *printFinder) Visit(n ast.Node) bool {
    if v.found { return false }
    if c, ok := n.(*ast.Call); ok {
        if name, ok := c.Func.(*ast.Name); ok && name.Ident == "print" {
            v.found = true
            return false
        }
    }
    return true
}

