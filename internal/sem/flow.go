package sem

import "functure/internal/ast"

// MustReturn checks if all paths in a function must return.
func MustReturn(stmts []ast.Stmt) bool {
    for _, stmt := range stmts {
        switch s := stmt.(type) {
        case *ast.ReturnStmt:
            return true
        case *ast.IfStmt:
            then := MustReturn(s.Then)
            elseBranch := MustReturn(s.Else)
            if then && elseBranch {
                return true
            }
        case *ast.WhileStmt, *ast.ForStmt:
            // Loops may not terminate
            continue
        }
    }
    return false
}
