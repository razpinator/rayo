// Package lint implements Rayo's readability- and correctness-focused linter.
//
// Each rule has a stable identifier (RYNNN) so that findings can be referenced,
// suppressed, and documented independently of their message text. Rules walk
// the AST (reusing the semantic analyzer's scope/type machinery where useful)
// and report structured findings with a source span, severity, and an optional
// autofix suggestion.
//
// Rule catalogue:
//
//	RY001  unused-binding        A variable is declared/assigned but never read.
//	RY002  unused-import         An imported module is never referenced.
//	RY003  suspicious-attr       obj.attr on a value that looks dict-like; the
//	                             author likely meant obj["attr"].
//	RY004  unsafe-optional-deref A possibly-None value is dereferenced (.attr or
//	                             [idx]) without a null check.
//	RY005  high-complexity       A function's cyclomatic complexity exceeds the
//	                             configured threshold.
package lint

import (
	"fmt"
	"sort"

	"rayo/internal/ast"
	"rayo/internal/diag"
	"rayo/internal/sem"
)

// Rule identifiers. Keep these stable; they are part of the linter's public
// contract (used in messages, docs, and any future suppression syntax).
const (
	RuleUnusedBinding     = "RY001"
	RuleUnusedImport      = "RY002"
	RuleSuspiciousAttr    = "RY003"
	RuleUnsafeOptionalRef = "RY004"
	RuleHighComplexity    = "RY005"
)

// Config tunes lint behaviour. The zero value is not recommended; use
// DefaultConfig for sensible thresholds.
type Config struct {
	// ComplexityThreshold is the maximum cyclomatic complexity a function may
	// have before RY005 fires. Functions at or below the threshold are fine.
	ComplexityThreshold int
}

// DefaultConfig returns the standard linter configuration.
func DefaultConfig() Config {
	return Config{ComplexityThreshold: 10}
}

// LintResult is a single finding.
type LintResult struct {
	// RuleID is the stable rule identifier (e.g. "RY001").
	RuleID string
	// Severity classifies the finding.
	Severity diag.Severity
	// Msg is a human-readable description.
	Msg string
	// Span locates the finding in source (may be zero if unknown).
	Span diag.Span
	// Fix is an optional autofix suggestion; empty when none applies.
	Fix string
}

// String renders a finding as "RY001 [warning] message" for CLI output.
func (r LintResult) String() string {
	return fmt.Sprintf("%s [%s] %s", r.RuleID, r.Severity, r.Msg)
}

// LintModule runs all lint rules on a module with the default configuration.
func LintModule(mod *ast.Module) []LintResult {
	return LintModuleWith(mod, DefaultConfig())
}

// LintModuleWith runs all lint rules with an explicit configuration. Findings
// are returned sorted by source position (then rule ID) for deterministic
// output.
func LintModuleWith(mod *ast.Module, cfg Config) []LintResult {
	l := &linter{cfg: cfg}
	l.checkImports(mod)
	// Analyze the top-level scope and recurse into functions/blocks.
	l.checkBlock(mod.Body, sem.NewScope(nil), true)
	l.sortResults()
	return l.results
}

type linter struct {
	cfg     Config
	results []LintResult
}

func (l *linter) add(r LintResult) { l.results = append(l.results, r) }

func (l *linter) sortResults() {
	sort.SliceStable(l.results, func(i, j int) bool {
		a, b := l.results[i], l.results[j]
		if a.Span.Start.Offset != b.Span.Start.Offset {
			return a.Span.Start.Offset < b.Span.Start.Offset
		}
		return a.RuleID < b.RuleID
	})
}

// checkImports flags imported modules that are never referenced (RY002). A
// module named "rayo/stdlib/data" is considered used if the identifier "data"
// (its last path segment) appears anywhere in the body.
func (l *linter) checkImports(mod *ast.Module) {
	if len(mod.Imports) == 0 {
		return
	}
	used := collectUsedNames(mod.Body)
	for _, imp := range mod.Imports {
		alias := importAlias(imp.Path)
		if alias == "" {
			continue
		}
		if !used[alias] {
			l.add(LintResult{
				RuleID:   RuleUnusedImport,
				Severity: diag.SeverityWarning,
				Msg:      fmt.Sprintf("unused import: %q (no reference to %q)", imp.Path, alias),
				Span:     imp.Span(),
				Fix:      "remove the unused import",
			})
		}
	}
}

// checkBlock lints a statement list in its own scope. When topLevel is true the
// scope is the module scope (used for unused-binding reporting after the walk).
func (l *linter) checkBlock(stmts []ast.Stmt, scope *sem.Scope, reportUnused bool) {
	declared := map[string]diag.Span{}
	used := map[string]bool{}

	// First pass: record declarations and expression-level rules.
	for _, stmt := range stmts {
		l.checkStmt(stmt, scope, declared, used)
	}

	if reportUnused {
		for name, span := range declared {
			if !used[name] {
				l.add(LintResult{
					RuleID:   RuleUnusedBinding,
					Severity: diag.SeverityWarning,
					Msg:      "unused variable: " + name,
					Span:     span,
					Fix:      "remove the declaration of " + name,
				})
			}
		}
	}
}

func (l *linter) checkStmt(stmt ast.Stmt, scope *sem.Scope, declared map[string]diag.Span, used map[string]bool) {
	switch s := stmt.(type) {
	case *ast.FuncDef:
		l.checkFuncDef(s, scope)

	case *ast.VarStmt:
		scope.Symbols[s.Name] = sem.InferType(s.Value)
		declared[s.Name] = s.Span()
		if s.Value != nil {
			l.checkExpr(s.Value, scope, used)
		}

	case *ast.AssignStmt:
		if name, ok := s.Target.(*ast.Name); ok {
			if _, seen := declared[name.Ident]; !seen {
				declared[name.Ident] = s.Span()
			}
			scope.Symbols[name.Ident] = sem.InferType(s.Value)
		} else {
			l.checkExpr(s.Target, scope, used)
		}
		if s.Value != nil {
			l.checkExpr(s.Value, scope, used)
		}

	case *ast.IfStmt:
		l.checkExpr(s.Cond, scope, used)
		l.checkBlock(s.Then, sem.NewScope(scope), true)
		for _, elif := range s.Elifs {
			l.checkExpr(elif.Cond, scope, used)
			l.checkBlock(elif.Body, sem.NewScope(scope), true)
		}
		l.checkBlock(s.Else, sem.NewScope(scope), true)

	case *ast.WhileStmt:
		l.checkExpr(s.Cond, scope, used)
		l.checkBlock(s.Body, sem.NewScope(scope), true)

	case *ast.ForStmt:
		child := sem.NewScope(scope)
		child.Symbols[s.Var] = &sem.AnyType{}
		l.checkExpr(s.Iter, scope, used)
		l.checkBlock(s.Body, child, true)

	case *ast.ReturnStmt:
		if s.Value != nil {
			l.checkExpr(s.Value, scope, used)
		}

	case *ast.TryStmt:
		l.checkBlock(s.Body, sem.NewScope(scope), true)
		for _, exc := range s.Excepts {
			child := sem.NewScope(scope)
			if exc.Var != "" {
				child.Symbols[exc.Var] = &sem.AnyType{}
			}
			l.checkBlock(exc.Body, child, true)
		}
		l.checkBlock(s.Finally, sem.NewScope(scope), true)

	case *ast.ExprStmt:
		l.checkExpr(s.Expr, scope, used)
	}
}

// checkFuncDef lints a function body in a fresh scope and computes its
// cyclomatic complexity (RY005).
func (l *linter) checkFuncDef(fn *ast.FuncDef, parent *sem.Scope) {
	scope := sem.NewScope(parent)
	for _, p := range fn.Params {
		scope.Symbols[p.Name] = &sem.AnyType{}
	}
	l.checkBlock(fn.Body, scope, true)

	if c := cyclomaticComplexity(fn.Body); c > l.cfg.ComplexityThreshold {
		l.add(LintResult{
			RuleID:   RuleHighComplexity,
			Severity: diag.SeverityWarning,
			Msg:      fmt.Sprintf("high cyclomatic complexity in %q: %d (threshold %d)", fn.Name, c, l.cfg.ComplexityThreshold),
			Span:     fn.Span(),
			Fix:      "extract helper functions to reduce branching",
		})
	}
}

// checkExpr walks an expression, marking name uses and running the attr/index
// rules (RY003, RY004).
func (l *linter) checkExpr(expr ast.Expr, scope *sem.Scope, used map[string]bool) {
	switch e := expr.(type) {
	case *ast.Name:
		used[e.Ident] = true

	case *ast.Attr:
		l.checkExpr(e.Target, scope, used)
		l.checkMemberAccess(e.Target, e.Attr, e.Span(), scope)

	case *ast.Index:
		l.checkExpr(e.Target, scope, used)
		l.checkExpr(e.Index, scope, used)
		if sem.IsOptional(sem.InferTypeIn(e.Target, scope)) {
			l.add(LintResult{
				RuleID:   RuleUnsafeOptionalRef,
				Severity: diag.SeverityWarning,
				Msg:      "possible unsafe index of optional value",
				Span:     e.Span(),
				Fix:      "check for None before indexing",
			})
		}

	case *ast.Call:
		l.checkExpr(e.Func, scope, used)
		for _, arg := range e.Args {
			l.checkExpr(arg, scope, used)
		}

	case *ast.BinaryOp:
		l.checkExpr(e.Left, scope, used)
		l.checkExpr(e.Right, scope, used)

	case *ast.UnaryOp:
		l.checkExpr(e.Right, scope, used)

	case *ast.DictLit:
		for i := range e.Keys {
			l.checkExpr(e.Keys[i], scope, used)
			l.checkExpr(e.Vals[i], scope, used)
		}

	case *ast.ListLit:
		for _, elem := range e.Elems {
			l.checkExpr(elem, scope, used)
		}

	case *ast.Lambda:
		child := sem.NewScope(scope)
		for _, p := range e.Params {
			child.Symbols[p.Name] = &sem.AnyType{}
		}
		l.checkExpr(e.Body, child, used)
	}
}

// checkMemberAccess applies RY004 (unsafe optional deref) and RY003 (suspicious
// attribute access on a dict-like value).
func (l *linter) checkMemberAccess(target ast.Expr, attr string, span diag.Span, scope *sem.Scope) {
	typ := sem.InferTypeIn(target, scope)

	// RY004: dereferencing an optional without a check.
	if sem.IsOptional(typ) {
		l.add(LintResult{
			RuleID:   RuleUnsafeOptionalRef,
			Severity: diag.SeverityWarning,
			Msg:      "possible unsafe dereference of optional value",
			Span:     span,
			Fix:      "check for None before dereferencing (obj?.attr or if obj != None)",
		})
		return
	}

	// RY003: attribute access on a value that is statically dict-like. A dict
	// literal target, or a value inferred as `dict`, is the strongest signal.
	if isDictLike(target, typ) {
		l.add(LintResult{
			RuleID:   RuleSuspiciousAttr,
			Severity: diag.SeverityInfo,
			Msg:      fmt.Sprintf("suspicious attribute access .%s on dict-like value", attr),
			Span:     span,
			Fix:      fmt.Sprintf("use obj[%q] instead of obj.%s", attr, attr),
		})
	}
}

// isDictLike reports whether target is likely a dict, so that member access via
// .attr is suspicious. A dict literal or an inferred `dict` type qualifies. An
// `any` value alone is too weak a signal (it produces noise), so it does not.
func isDictLike(target ast.Expr, typ sem.Type) bool {
	if _, ok := target.(*ast.DictLit); ok {
		return true
	}
	if bt, ok := typ.(*sem.BasicType); ok && bt.Name == "dict" {
		return true
	}
	return false
}

// cyclomaticComplexity computes an approximate cyclomatic complexity for a
// function body: 1 plus one for each decision point (if, elif, while, for,
// except handler) and each boolean operator (and/or) in conditions.
func cyclomaticComplexity(stmts []ast.Stmt) int {
	v := &complexityVisitor{count: 1}
	for _, s := range stmts {
		ast.Walk(v, s)
	}
	return v.count
}

type complexityVisitor struct{ count int }

func (v *complexityVisitor) Visit(n ast.Node) bool {
	switch node := n.(type) {
	case *ast.FuncDef:
		// Nested function complexity is measured separately.
		return false
	case *ast.IfStmt:
		v.count++
		v.count += len(node.Elifs)
	case *ast.WhileStmt, *ast.ForStmt:
		v.count++
	case *ast.TryStmt:
		v.count += len(node.Excepts)
	case *ast.BinaryOp:
		if node.Op == "and" || node.Op == "or" || node.Op == "&&" || node.Op == "||" {
			v.count++
		}
	}
	return true
}

// collectUsedNames returns the set of identifier names referenced anywhere in
// stmts, used to detect unused imports.
func collectUsedNames(stmts []ast.Stmt) map[string]bool {
	v := &nameCollector{names: map[string]bool{}}
	for _, s := range stmts {
		ast.Walk(v, s)
	}
	return v.names
}

type nameCollector struct{ names map[string]bool }

func (v *nameCollector) Visit(n ast.Node) bool {
	if name, ok := n.(*ast.Name); ok {
		v.names[name.Ident] = true
	}
	return true
}

// importAlias derives the identifier an import path is referenced by: the last
// path segment. "rayo/stdlib/data" -> "data".
func importAlias(path string) string {
	// Trim surrounding quotes if the parser preserved them.
	p := path
	if len(p) >= 2 && (p[0] == '"' || p[0] == '\'') {
		p = p[1 : len(p)-1]
	}
	last := ""
	seg := ""
	for _, r := range p {
		if r == '/' || r == '.' {
			if seg != "" {
				last = seg
			}
			seg = ""
			continue
		}
		seg += string(r)
	}
	if seg != "" {
		last = seg
	}
	return last
}
