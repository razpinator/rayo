package gen

import (
	"strings"

	"rayo/internal/ast"
)

// LowerTryExcept lowers try/except/finally to a Go statement string using
// recover-based error handling. EmitStmt handles TryStmt inline via emitTry;
// this helper exposes the same lowering to callers that want the generated Go
// as a string (e.g. tests, tooling).
func LowerTryExcept(try *ast.TryStmt) string {
	ctx := &GenContext{PackageName: "main", Code: &strings.Builder{}, declared: map[string]bool{}}
	emitTry(try, ctx)
	return ctx.Code.String()
}
