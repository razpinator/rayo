package sem

import (
    "functure/internal/ast"
    "functure/internal/diag"
)

// Scope represents a lexical scope.
type Scope struct {
    Parent *Scope
    Symbols map[string]Type
    Used    map[string]bool
}

func NewScope(parent *Scope) *Scope {
    return &Scope{Parent: parent, Symbols: map[string]Type{}, Used: map[string]bool{}}
}

// Reporter for diagnostics
var reporter diag.Reporter

// CheckModule performs semantic checks on a module.
func CheckModule(mod *ast.Module, rep diag.Reporter) {
    reporter = rep
    scope := NewScope(nil)
    for _, stmt := range mod.Body {
        checkStmt(stmt, scope)
    }
    // Warn for unused vars
    for name, used := range scope.Used {
        if !used {
            reporter.Report(diag.Span{}, "unused variable: "+name)
        }
    }
}

func checkStmt(stmt ast.Stmt, scope *Scope) {
    switch s := stmt.(type) {
    case *ast.VarStmt:
        typ := ast.Any{}
        if s.Value != nil {
            typ = ast.Any{} // Could use InferType(s.Value)
        }
        scope.Symbols[s.Name] = typ
        scope.Used[s.Name] = false
    case *ast.AssignStmt:
        // Mark as used
        if name, ok := s.Target.(*ast.Name); ok {
            scope.Used[name.Ident] = true
        }
    case *ast.IfStmt:
        for _, stmt := range s.Then {
            checkStmt(stmt, scope)
        }
        for _, elif := range s.Elifs {
            for _, stmt := range elif.Body {
                checkStmt(stmt, scope)
            }
        }
        for _, stmt := range s.Else {
            checkStmt(stmt, scope)
        }
    case *ast.WhileStmt:
        for _, stmt := range s.Body {
            checkStmt(stmt, scope)
        }
    case *ast.ForStmt:
        scope.Symbols[s.Var] = ast.Any{}
        scope.Used[s.Var] = false
        for _, stmt := range s.Body {
            checkStmt(stmt, scope)
        }
    case *ast.ReturnStmt:
        // Could check return type
    case *ast.TryStmt:
        for _, stmt := range s.Body {
            checkStmt(stmt, scope)
        }
        for _, exc := range s.Excepts {
            for _, stmt := range exc.Body {
                checkStmt(stmt, scope)
            }
        }
        for _, stmt := range s.Finally {
            checkStmt(stmt, scope)
        }
    case *ast.ExprStmt:
        // Could check expression
    }
}
