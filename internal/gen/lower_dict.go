package gen

import (
    "functure/internal/ast"
    "fmt"
)

// LowerDict lowers dict literals to Go map[string]any.
func LowerDict(dict *ast.DictLit) string {
    code := "map[string]any{"
    for i := range dict.Keys {
        code += fmt.Sprintf("%v: %v, ", dict.Keys[i], dict.Vals[i])
    }
    code += "}"
    return code
}
