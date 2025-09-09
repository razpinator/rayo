package parse

import (
    "functure/internal/ast"
    "fmt"
    "strings"
)

// PrettyPrint reconstructs source-like code from AST for debugging.
func PrettyPrint(n ast.Node) string {
    var sb strings.Builder
    ppNode(&sb, n, 0)
    return sb.String()
}

func ppNode(sb *strings.Builder, n ast.Node, indent int) {
    if n == nil {
        return
    }
    pad := strings.Repeat("  ", indent)
    switch x := n.(type) {
    case *ast.Module:
        sb.WriteString(fmt.Sprintf("module %s {\n", x.Name))
        for _, imp := range x.Imports {
            ppNode(sb, imp, indent+1)
        }
        for _, stmt := range x.Body {
            ppNode(sb, stmt, indent+1)
        }
        sb.WriteString("}\n")
    case *ast.Import:
        sb.WriteString(pad + fmt.Sprintf("import '%s'\n", x.Path))
    case *ast.VarStmt:
        sb.WriteString(pad + fmt.Sprintf("var %s = ...\n", x.Name))
    case *ast.IfStmt:
        sb.WriteString(pad + "if ... {\n")
        for _, stmt := range x.Then {
            ppNode(sb, stmt, indent+1)
        }
        sb.WriteString(pad + "}\n")
    case *ast.ReturnStmt:
        sb.WriteString(pad + "return ...\n")
    // ...extend for other node types...
    }
}
