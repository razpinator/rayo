package sem

import "rayo/internal/ast"

// Type system for semantic analysis.
//
// Rayo's spec (docs/spec.md) defines a small, learner-friendly type system:
// basic types (int, float, str, bool), optional types (T?), and the dynamic
// escape hatch `any`. The checker infers types where it can so that
// null-safety rules ("None must be explicitly handled") can be enforced.

// Type is the interface implemented by all inferred types.
type Type interface{ typeName() string }

// BasicType is a concrete scalar or named type: int, float, str, bool, dict,
// list, or a user-defined identifier.
type BasicType struct {
	Name string
}

func (b *BasicType) typeName() string { return b.Name }

// OptionalType is `Elem?`: a value that may be the concrete Elem or None.
type OptionalType struct {
	Elem Type
}

func (o *OptionalType) typeName() string {
	if o.Elem == nil {
		return "any?"
	}
	return o.Elem.typeName() + "?"
}

// AnyType is the dynamic type used when inference cannot determine a concrete
// type. It participates in no null-safety checks.
type AnyType struct{}

func (a *AnyType) typeName() string { return "any" }

// TypeString renders a type for diagnostics. nil is treated as `any`.
func TypeString(t Type) string {
	if t == nil {
		return "any"
	}
	return t.typeName()
}

// IsOptional reports whether t is an optional type (T?).
func IsOptional(t Type) bool {
	_, ok := t.(*OptionalType)
	return ok
}

// resolver is any object that can look up the inferred type of a name in the
// current lexical scope. *Scope implements it. It is passed to inferType so
// inference can consult the symbol table instead of always returning `any`.
type resolver interface {
	lookup(name string) (Type, bool)
}

// InferType infers the static type of expr without a symbol table. Names and
// member accesses that require scope resolution fall back to `any`. This is the
// stable entry point used by tools (REPL `:type`, linter). Use InferTypeIn when
// a scope is available for precise, null-safety-aware inference.
func InferType(expr ast.Expr) Type {
	return inferType(expr, nil)
}

// InferTypeIn infers the static type of expr, resolving names and members
// through scope so that optional values flow through the expression tree.
func InferTypeIn(expr ast.Expr, scope *Scope) Type {
	if scope == nil {
		return inferType(expr, nil)
	}
	return inferType(expr, scope)
}

func inferType(expr ast.Expr, res resolver) Type {
	switch e := expr.(type) {
	case *ast.Literal:
		switch e.Value.(type) {
		case int, int64:
			return &BasicType{Name: "int"}
		case float64, float32:
			return &BasicType{Name: "float"}
		case string:
			return &BasicType{Name: "str"}
		case bool:
			return &BasicType{Name: "bool"}
		case nil:
			// A bare None literal is optional-of-unknown.
			return &OptionalType{Elem: &AnyType{}}
		default:
			return &AnyType{}
		}
	case *ast.Name:
		if res != nil {
			if t, ok := res.lookup(e.Ident); ok {
				return t
			}
		}
		return &AnyType{}
	case *ast.DictLit:
		return &BasicType{Name: "dict"}
	case *ast.ListLit:
		return &BasicType{Name: "list"}
	case *ast.UnaryOp:
		if e.Op == "not" {
			return &BasicType{Name: "bool"}
		}
		return inferType(e.Right, res)
	case *ast.BinaryOp:
		return inferBinary(e, res)
	case *ast.Attr:
		// Member access on a known optional stays dynamic but is *not* itself
		// optional once dereferenced; the null-safety pass flags the unsafe
		// dereference separately. Unknown targets stay dynamic.
		return &AnyType{}
	case *ast.Index:
		return &AnyType{}
	case *ast.Call:
		// `.get(key)` style lookups conventionally yield an optional in Rayo
		// (docs/spec.md: "scores.get(...) // Optional method"). Model that so
		// callers must handle the None case.
		if attr, ok := e.Func.(*ast.Attr); ok && attr.Attr == "get" && len(e.Args) == 1 {
			return &OptionalType{Elem: &AnyType{}}
		}
		return &AnyType{}
	default:
		return &AnyType{}
	}
}

// inferBinary infers the result type of a binary operation. Comparisons and
// logical operators produce bool; `+`, `-`, `*`, `/` propagate a numeric or
// string type when both operands agree, otherwise stay dynamic.
func inferBinary(e *ast.BinaryOp, res resolver) Type {
	switch e.Op {
	case "==", "!=", "<", ">", "<=", ">=", "and", "or":
		// `or` is Rayo's null-coalescing/logical operator; its result is only
		// bool for comparisons. For `or` used as a default (a or b) we keep it
		// dynamic below.
		if e.Op == "or" {
			break
		}
		return &BasicType{Name: "bool"}
	}
	lt := inferType(e.Left, res)
	rt := inferType(e.Right, res)
	// `a or b` yields the non-optional fallback type when the right side is a
	// concrete type; this is how safe navigation with a default is typed.
	if e.Op == "or" {
		if bt, ok := rt.(*BasicType); ok {
			return bt
		}
		return &AnyType{}
	}
	if lb, ok := lt.(*BasicType); ok {
		if rb, ok := rt.(*BasicType); ok && lb.Name == rb.Name {
			return &BasicType{Name: lb.Name}
		}
	}
	return &AnyType{}
}
