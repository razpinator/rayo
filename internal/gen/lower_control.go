package gen

import (
    "rayo/internal/ast"
)

// LowerTryExcept lowers try/except/finally to Go error handling.
func LowerTryExcept(try *ast.TryStmt) string {
    // Example: lower to Go function returning (T, error)
    code := "func() (any, error) {\n"
    code += "  defer func() { /* finally */ }()\n"
    code += "  // try block\n"
    code += "  // except handlers\n"
    code += "  return nil, nil\n"
    code += "}()"
    return code
}
