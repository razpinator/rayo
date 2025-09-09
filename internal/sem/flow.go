package sem

import "functure/internal/ast"

// MustReturn checks if all paths in a function must return.
func MustReturn(stmts []ast.Stmt) bool {
    mustReturn := false
    for _, stmt := range stmts {
        switch s := stmt.(type) {
        case *ast.ReturnStmt:
            mustReturn = true
        case *ast.IfStmt:
            then := MustReturn(s.Then)
            elseBranch := MustReturn(s.Else)
            if then && elseBranch {
                mustReturn = true
            }
        case *ast.WhileStmt, *ast.ForStmt:
            // Loops may not terminate
            continue
        case *ast.TryStmt:
            body := MustReturn(s.Body)
            finally := MustReturn(s.Finally)
            if body && finally {
                mustReturn = true
            }
        }
    }
    return mustReturn
}
