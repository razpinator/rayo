package ast

// Visitor interface for AST traversal.
type Visitor interface {
	Visit(Node) bool // return false to skip children
}

// Walk traverses the AST node and its children.
func Walk(v Visitor, n Node) {
	if n == nil || !v.Visit(n) {
		return
	}
	switch x := n.(type) {
	case *Module:
		for _, imp := range x.Imports {
			Walk(v, imp)
		}
		for _, stmt := range x.Body {
			Walk(v, stmt)
		}
	case *Import:
		// no children
	case *FuncDef:
		for _, d := range x.Decorators {
			Walk(v, d)
		}
		for _, p := range x.Params {
			Walk(v, p)
		}
		for _, stmt := range x.Body {
			Walk(v, stmt)
		}
	case *ClassDef:
		for _, f := range x.Fields {
			Walk(v, f)
		}
		for _, m := range x.Methods {
			Walk(v, m)
		}
		for _, p := range x.Properties {
			Walk(v, p)
		}
	case *Field:
		// no children
	case *Property:
		for _, stmt := range x.Get {
			Walk(v, stmt)
		}
		for _, stmt := range x.Set {
			Walk(v, stmt)
		}
	case *Param:
		// no children
	case Stmt:
		switch s := x.(type) {
		case *VarStmt:
			Walk(v, s.Value)
		case *AssignStmt:
			Walk(v, s.Target)
			Walk(v, s.Value)
		case *IfStmt:
			Walk(v, s.Cond)
			for _, stmt := range s.Then {
				Walk(v, stmt)
			}
			for _, elif := range s.Elifs {
				Walk(v, elif)
			}
			for _, stmt := range s.Else {
				Walk(v, stmt)
			}
		case *Elif:
			Walk(v, s.Cond)
			for _, stmt := range s.Body {
				Walk(v, stmt)
			}
		case *WhileStmt:
			Walk(v, s.Cond)
			for _, stmt := range s.Body {
				Walk(v, stmt)
			}
		case *ForStmt:
			Walk(v, s.Iter)
			for _, stmt := range s.Body {
				Walk(v, stmt)
			}
		case *ReturnStmt:
			Walk(v, s.Value)
		case *TryStmt:
			for _, stmt := range s.Body {
				Walk(v, stmt)
			}
			for _, exc := range s.Excepts {
				Walk(v, exc)
			}
			for _, stmt := range s.Finally {
				Walk(v, stmt)
			}
		case *Except:
			for _, stmt := range s.Body {
				Walk(v, stmt)
			}
		case *ExprStmt:
			Walk(v, s.Expr)
		case *MatchStmt:
			Walk(v, s.Subject)
			for _, cs := range s.Cases {
				if cs.Pattern != nil {
					Walk(v, cs.Pattern)
				}
				for _, stmt := range cs.Body {
					Walk(v, stmt)
				}
			}
		}
	case Expr:
		switch e := x.(type) {
		case *Literal:
			// no children
		case *Name:
			// no children
		case *Call:
			Walk(v, e.Func)
			for _, arg := range e.Args {
				Walk(v, arg)
			}
		case *Index:
			Walk(v, e.Target)
			Walk(v, e.Index)
		case *Attr:
			Walk(v, e.Target)
		case *UnaryOp:
			Walk(v, e.Right)
		case *BinaryOp:
			Walk(v, e.Left)
			Walk(v, e.Right)
		case *DictLit:
			for i := range e.Keys {
				Walk(v, e.Keys[i])
				Walk(v, e.Vals[i])
			}
		case *ListLit:
			for _, elem := range e.Elems {
				Walk(v, elem)
			}
		case *Lambda:
			for _, p := range e.Params {
				Walk(v, p)
			}
			Walk(v, e.Body)
		}
	}
}
