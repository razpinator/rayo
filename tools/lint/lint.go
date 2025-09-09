package lint

import (
    "functure/internal/ast"
    "functure/internal/sem"
)

type LintResult struct {
    Msg string
    Fix string // autofix suggestion
}

// LintModule runs all lint rules on a module.
func LintModule(mod *ast.Module) []LintResult {
    var results []LintResult
    // Unused variables
    scope := sem.NewScope(nil)
    for _, stmt := range mod.Body {
        if v, ok := stmt.(*ast.VarStmt); ok {
            scope.Symbols[v.Name] = ast.Any{}
            scope.Used[v.Name] = false
        }
        if a, ok := stmt.(*ast.AssignStmt); ok {
            if name, ok := a.Target.(*ast.Name); ok {
                scope.Used[name.Ident] = true
            }
        }
    }
    for name, used := range scope.Used {
        if !used {
            results = append(results, LintResult{
                Msg: "unused variable: " + name,
                Fix: "remove declaration of " + name,
            })
        }
    }
    // Suspicious attr vs key (simple heuristic)
    for _, stmt := range mod.Body {
        if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
            if attr, ok := exprStmt.Expr.(*ast.Attr); ok {
                typ := sem.InferType(attr.Target)
                if _, isDict := typ.(*sem.AnyType); isDict {
                    results = append(results, LintResult{
                        Msg: "suspicious attribute access on dict-like object",
                        Fix: "use obj[\"attr\"] instead of obj.attr",
                    })
                }
            }
        }
    }
    // Null-safety hints
    for _, stmt := range mod.Body {
        if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
            if attr, ok := exprStmt.Expr.(*ast.Attr); ok {
                typ := sem.InferType(attr.Target)
                if _, ok := typ.(*sem.OptionalType); ok {
                    results = append(results, LintResult{
                        Msg: "possible unsafe dereference of optional value",
                        Fix: "add null check before dereference",
                    })
                }
            }
        }
    }
    // Cyclomatic complexity (simple count of branches)
    complexity := 1
    for _, stmt := range mod.Body {
        switch stmt.(type) {
        case *ast.IfStmt, *ast.WhileStmt, *ast.ForStmt, *ast.TryStmt:
            complexity++
        }
    }
    if complexity > 10 {
        results = append(results, LintResult{
            Msg: "high cyclomatic complexity: %d", // can format with fmt.Sprintf
            Fix: "refactor to reduce branches",
        })
    }
    return results
}
