// Package opt implements AST-to-AST optimization passes that run between
// semantic analysis and code generation. All passes are behavior-preserving.
//
// Pass pipeline (see Optimize):
//
//  1. constant folding      — evaluate literal expressions (fold.go)
//  2. small-function inlining — inline trivial single-return calls (inline.go)
//  3. constant folding      — fold expressions exposed by inlining
//  4. dead code elimination — drop unreachable statements, prune constant
//     branches (dce.go)
//
// A note on "escape-oriented pointer choices" (a planned optimization): Rayo
// transpiles to Go, and the Go compiler already performs escape analysis on the
// generated code to decide stack-vs-heap allocation. Re-deriving those choices
// in the transpiler would duplicate work the Go toolchain does better and could
// only pessimize it, so this stage is intentionally delegated to `go build`
// rather than implemented as a Rayo pass. The transpiler's responsibility is to
// emit idiomatic Go (value receivers/returns where natural, pointers only for
// constructed instances), which it does; escape analysis then runs downstream.
package opt

import "rayo/internal/ast"

// Optimize runs the full behavior-preserving optimization pipeline over the
// module in place and returns it.
func Optimize(mod *ast.Module) *ast.Module {
	if mod == nil {
		return nil
	}
	FoldModule(mod)
	InlineModule(mod)
	FoldModule(mod)
	EliminateDeadCode(mod)
	return mod
}
