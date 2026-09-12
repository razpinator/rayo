package gen

import (
	"fmt"
	"strings"

	"rayo/internal/ast"
)

// LowerDict lowers a dict literal to a Go map[string]any. Keys and values are
// emitted through emitExpr so nested expressions are rendered as code (the
// previous implementation printed raw AST pointers via %v).
func LowerDict(dict *ast.DictLit, ctx *GenContext) string {
	var b strings.Builder
	b.WriteString("map[string]any{")
	for i := range dict.Keys {
		key := emitDictKey(dict.Keys[i], ctx)
		b.WriteString(fmt.Sprintf("%s: %s", key, emitExpr(dict.Vals[i], ctx)))
		if i < len(dict.Keys)-1 {
			b.WriteString(", ")
		}
	}
	b.WriteString("}")
	return b.String()
}

// emitDictKey renders a dict key. String-literal keys become Go string
// literals; any other expression is emitted as-is and expected to be a string.
func emitDictKey(k ast.Expr, ctx *GenContext) string {
	if lit, ok := k.(*ast.Literal); ok {
		if s, ok := lit.Value.(string); ok {
			return fmt.Sprintf("%q", s)
		}
	}
	return emitExpr(k, ctx)
}

// LowerList lowers a list literal to a Go []any slice literal.
func LowerList(list *ast.ListLit, ctx *GenContext) string {
	elems := make([]string, len(list.Elems))
	for i, el := range list.Elems {
		elems[i] = emitExpr(el, ctx)
	}
	return fmt.Sprintf("[]any{%s}", strings.Join(elems, ", "))
}
