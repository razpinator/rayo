package gen

import (
	"fmt"
	"strings"

	"rayo/internal/ast"
)

// EmitModule emits Go code for a module AST. It is a convenience entry point
// used by tests and benchmarks; the multi-file compiler drives EmitStmt
// directly (see internal/compile). The `fmt` import is added only when the
// module actually calls print(), keeping generated code go-vet clean.
func EmitModule(mod *ast.Module, ctx *GenContext) string {
	ctx.Code.WriteString(fmt.Sprintf("package %s\n\n", ctx.PackageName))

	hasPrint := false
	for _, stmt := range mod.Body {
		if ContainsPrint(stmt) {
			hasPrint = true
			break
		}
	}
	imported := map[string]bool{}
	for _, imp := range mod.Imports {
		imported[imp.Path] = true
	}
	if hasPrint && !imported["fmt"] {
		ctx.Code.WriteString("import \"fmt\"\n")
	}
	for _, imp := range mod.Imports {
		if imp.Alias != "" {
			ctx.Code.WriteString(fmt.Sprintf("import %s %q\n", imp.Alias, imp.Path))
		} else {
			ctx.Code.WriteString(fmt.Sprintf("import %q\n", imp.Path))
		}
	}

	// Emit top-level function definitions first, then wrap the remaining
	// statements in main() so `return` at top level is valid Go.
	if ctx.declared == nil {
		ctx.declared = map[string]bool{}
	}
	RegisterClasses(mod.Body, ctx)
	var funcs, body strings.Builder
	for _, stmt := range mod.Body {
		target := &body
		if IsTopLevelDecl(stmt) {
			target = &funcs
		}
		sub := &GenContext{PackageName: ctx.PackageName, Code: target, TempVarIdx: ctx.TempVarIdx, declared: ctx.declared, classes: ctx.classes}
		EmitStmt(stmt, sub)
		ctx.TempVarIdx = sub.TempVarIdx
	}
	ctx.Code.WriteString(funcs.String())
	if body.Len() > 0 {
		ctx.Code.WriteString("func main() {\n")
		ctx.Code.WriteString(body.String())
		ctx.Code.WriteString("}\n")
	}
	return ctx.Code.String()
}

// RegisterClasses pre-registers every class and top-level function declared in
// stmts so constructor calls resolve to `New<Class>` and calls to a top-level
// function are emitted directly (while calls to dynamic `any` values go through
// a callable type assertion), regardless of declaration order.
func RegisterClasses(stmts []ast.Stmt, ctx *GenContext) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.ClassDef:
			ctx.registerClass(s.Name)
		case *ast.FuncDef:
			ctx.registerFunc(s.Name)
		}
	}
}

// IsTopLevelDecl reports whether a statement must be emitted at Go file scope
// (functions, classes) rather than inside the synthetic main() body.
func IsTopLevelDecl(stmt ast.Stmt) bool {
	switch stmt.(type) {
	case *ast.FuncDef, *ast.ClassDef:
		return true
	}
	return false
}

// EmitStmt emits Go for a single statement.
func EmitStmt(stmt ast.Stmt, ctx *GenContext) {
	switch s := stmt.(type) {
	case *ast.FuncDef:
		emitFuncDef(s, ctx)

	case *ast.ClassDef:
		emitClassDef(s, ctx)

	case *ast.VarStmt:
		ctx.markDeclared(s.Name)
		ctx.Code.WriteString(fmt.Sprintf("var %s = %s\n", s.Name, emitExpr(s.Value, ctx)))
		// Reference the binding so an unused declaration does not break the Go
		// build; Rayo tolerates locally-unused bindings (they are warned, not
		// rejected, by the semantic checker).
		ctx.Code.WriteString(fmt.Sprintf("_ = %s\n", s.Name))

	case *ast.AssignStmt:
		// Plain-name targets declare with `:=` on first sight and reassign with
		// `=` afterwards. Non-name targets (index/attr) always use `=`.
		if name, ok := s.Target.(*ast.Name); ok {
			if ctx.markDeclared(name.Ident) {
				ctx.Code.WriteString(fmt.Sprintf("%s := %s\n", name.Ident, emitExpr(s.Value, ctx)))
				ctx.Code.WriteString(fmt.Sprintf("_ = %s\n", name.Ident))
			} else {
				ctx.Code.WriteString(fmt.Sprintf("%s = %s\n", name.Ident, emitExpr(s.Value, ctx)))
			}
		} else {
			ctx.Code.WriteString(fmt.Sprintf("%s = %s\n", emitExpr(s.Target, ctx), emitExpr(s.Value, ctx)))
		}

	case *ast.ExprStmt:
		ctx.Code.WriteString(fmt.Sprintf("%s\n", emitExpr(s.Expr, ctx)))

	case *ast.IfStmt:
		ctx.Code.WriteString(fmt.Sprintf("if %s {\n", emitExpr(s.Cond, ctx)))
		for _, st := range s.Then {
			EmitStmt(st, ctx)
		}
		for _, elif := range s.Elifs {
			ctx.Code.WriteString(fmt.Sprintf("} else if %s {\n", emitExpr(elif.Cond, ctx)))
			for _, st := range elif.Body {
				EmitStmt(st, ctx)
			}
		}
		if len(s.Else) > 0 {
			ctx.Code.WriteString("} else {\n")
			for _, st := range s.Else {
				EmitStmt(st, ctx)
			}
		}
		ctx.Code.WriteString("}\n")

	case *ast.WhileStmt:
		ctx.Code.WriteString(fmt.Sprintf("for %s {\n", emitExpr(s.Cond, ctx)))
		for _, st := range s.Body {
			EmitStmt(st, ctx)
		}
		ctx.Code.WriteString("}\n")

	case *ast.ForStmt:
		// `for x in iter { ... }` lowers to a Go range over the iterable.
		ctx.Code.WriteString(fmt.Sprintf("for _, %s := range %s {\n", s.Var, emitExpr(s.Iter, ctx)))
		ctx.Code.WriteString(fmt.Sprintf("_ = %s\n", s.Var))
		for _, st := range s.Body {
			EmitStmt(st, ctx)
		}
		ctx.Code.WriteString("}\n")

	case *ast.TryStmt:
		emitTry(s, ctx)

	case *ast.MatchStmt:
		emitMatch(s, ctx)

	case *ast.ReturnStmt:
		if s.Value != nil {
			ctx.Code.WriteString(fmt.Sprintf("return %s\n", emitExpr(s.Value, ctx)))
		} else {
			ctx.Code.WriteString("return nil\n")
		}
	}
}

// emitFuncDef emits a Go function. Non-generic, unannotated functions keep the
// dynamic shape `func Name(p any, ...) any` that the rest of the pipeline
// relies on. When the source declares generic type parameters (`def name[T]`)
// or type annotations, those are mapped to Go: type parameters become
// `[T any, ...]` constraints and annotated parameter/return types are lowered
// via goType so the generated code is genuinely generic and type-checked by Go.
func emitFuncDef(s *ast.FuncDef, ctx *GenContext) {
	// A decorated function is lowered to a package-level variable bound to the
	// decorators applied to a function literal:  var f = d1(d2(<literal>)).
	if len(s.Decorators) > 0 {
		emitDecoratedFunc(s, ctx)
		return
	}
	// Go requires `func main()` to take no arguments and return nothing. A
	// user-defined `def main()` maps to the program entry point, so emit it
	// with the required signature and no synthetic return.
	if s.Name == "main" && len(s.Params) == 0 && len(s.TypeParams) == 0 {
		ctx.Code.WriteString("func main() {\n")
		for _, bodyStmt := range s.Body {
			EmitStmt(bodyStmt, ctx)
		}
		ctx.Code.WriteString("}\n")
		return
	}

	typeParamSet := map[string]bool{}
	for _, tp := range s.TypeParams {
		typeParamSet[tp] = true
	}
	generic := len(s.TypeParams) > 0

	// Parameter list. Annotated params use their mapped Go type; unannotated
	// params stay `any` to preserve the dynamic default.
	params := make([]string, len(s.Params))
	for i, p := range s.Params {
		goT := "any"
		if p.Type != nil {
			goT = goType(p.Type, typeParamSet)
		}
		params[i] = p.Name + " " + goT
	}

	// Return type: an explicit annotation is honored; otherwise `any`.
	retT := "any"
	if s.RetType != nil {
		retT = goType(s.RetType, typeParamSet)
	}

	if generic {
		constraints := make([]string, len(s.TypeParams))
		for i, tp := range s.TypeParams {
			constraints[i] = tp + " any"
		}
		ctx.Code.WriteString(fmt.Sprintf("func %s[%s](%s) %s {\n",
			s.Name, strings.Join(constraints, ", "), strings.Join(params, ", "), retT))
	} else {
		ctx.Code.WriteString(fmt.Sprintf("func %s(%s) %s {\n", s.Name, strings.Join(params, ", "), retT))
	}

	for _, bodyStmt := range s.Body {
		EmitStmt(bodyStmt, ctx)
	}
	// Guarantee the function type-checks even when the body falls through
	// without an explicit return. A concrete non-`any` return type needs a
	// typed zero value.
	if !endsWithReturn(s.Body) {
		ctx.Code.WriteString(fmt.Sprintf("return %s\n", zeroValue(retT)))
	}
	ctx.Code.WriteString("}\n")
}

// emitClassDef lowers a Rayo class to Go: a struct for its fields, a
// constructor synthesized from `__init__`, receiver methods for the remaining
// methods, and getter/setter methods for each property.
//
// Everything is emitted into the single generated `package main`, so field and
// method names are used as written (unexported names are valid within one
// package). The receiver is always `self`, matching the source's first method
// parameter.
func emitClassDef(s *ast.ClassDef, ctx *GenContext) {
	ctx.registerClass(s.Name)
	// Struct definition.
	ctx.Code.WriteString(fmt.Sprintf("type %s struct {\n", s.Name))
	if s.Base != "" {
		// Embed the base struct for simple inheritance/field promotion.
		ctx.Code.WriteString(fmt.Sprintf("%s\n", s.Base))
	}
	for _, f := range s.Fields {
		goT := "any"
		if f.Type != nil {
			goT = goType(f.Type, nil)
		}
		ctx.Code.WriteString(fmt.Sprintf("%s %s\n", f.Name, goT))
	}
	ctx.Code.WriteString("}\n")

	// Methods and constructor.
	for _, m := range s.Methods {
		if m.Name == "__init__" {
			emitConstructor(s, m, ctx)
			continue
		}
		emitMethod(s, m, ctx)
	}

	// Properties -> getter/setter methods.
	for _, prop := range s.Properties {
		emitProperty(s, prop, ctx)
	}
}

// emitConstructor emits `func NewName(params) *Name { self := &Name{}; <body>; return self }`
// from a class's `__init__` method. The leading `self` parameter is dropped
// because it is created by the constructor.
func emitConstructor(s *ast.ClassDef, init *ast.FuncDef, ctx *GenContext) {
	params := methodParams(init) // excludes self
	ctx.Code.WriteString(fmt.Sprintf("func New%s(%s) *%s {\n", s.Name, strings.Join(params, ", "), s.Name))
	ctx.Code.WriteString(fmt.Sprintf("self := &%s{}\n", s.Name))
	ctx.Code.WriteString("_ = self\n")
	for _, st := range init.Body {
		EmitStmt(st, ctx)
	}
	ctx.Code.WriteString("return self\n")
	ctx.Code.WriteString("}\n")
}

// emitMethod emits a receiver method. The first parameter (`self`) becomes the
// receiver; the rest become the method parameters.
func emitMethod(s *ast.ClassDef, m *ast.FuncDef, ctx *GenContext) {
	params := methodParams(m)
	retT := "any"
	if m.RetType != nil {
		retT = goType(m.RetType, nil)
	}
	ctx.Code.WriteString(fmt.Sprintf("func (self *%s) %s(%s) %s {\n", s.Name, m.Name, strings.Join(params, ", "), retT))
	for _, st := range m.Body {
		EmitStmt(st, ctx)
	}
	if !endsWithReturn(m.Body) {
		ctx.Code.WriteString(fmt.Sprintf("return %s\n", zeroValue(retT)))
	}
	ctx.Code.WriteString("}\n")
}

// emitProperty emits a getter method named after the property and, when a
// setter body is present, a `SetName` method.
func emitProperty(s *ast.ClassDef, prop *ast.Property, ctx *GenContext) {
	retT := "any"
	if prop.Type != nil {
		retT = goType(prop.Type, nil)
	}
	// Getter.
	ctx.Code.WriteString(fmt.Sprintf("func (self *%s) %s() %s {\n", s.Name, prop.Name, retT))
	for _, st := range prop.Get {
		EmitStmt(st, ctx)
	}
	if !endsWithReturn(prop.Get) {
		ctx.Code.WriteString(fmt.Sprintf("return %s\n", zeroValue(retT)))
	}
	ctx.Code.WriteString("}\n")

	// Setter (optional).
	if len(prop.Set) > 0 {
		param := prop.SetParam
		if param == "" {
			param = "value"
		}
		ctx.Code.WriteString(fmt.Sprintf("func (self *%s) Set%s(%s %s) {\n", s.Name, capitalize(prop.Name), param, retT))
		for _, st := range prop.Set {
			EmitStmt(st, ctx)
		}
		ctx.Code.WriteString("}\n")
	}
}

// methodParams renders a method's parameters as Go, dropping the leading `self`
// receiver parameter. Annotated types are mapped; unannotated stay `any`.
func methodParams(m *ast.FuncDef) []string {
	var out []string
	for i, p := range m.Params {
		if i == 0 && p.Name == "self" {
			continue
		}
		goT := "any"
		if p.Type != nil {
			goT = goType(p.Type, nil)
		}
		out = append(out, p.Name+" "+goT)
	}
	return out
}

// capitalize upper-cases the first byte of s (ASCII), used to form SetName.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	b := []byte(s)
	if b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 'a' - 'A'
	}
	return string(b)
}

// emitDecoratedFunc lowers a decorated function to a package-level variable
// bound to the decorators applied to a function literal. Decorators are stored
// outermost-first (top-to-bottom in source), so for
//
//	@a
//	@b
//	def f(x) { ... }
//
// this emits `var f = a(b(func(x any) any { ... }))`, i.e. the decorator
// nearest the def (`b`) is applied first, matching Python semantics. The
// variable is also referenced with `_ = f` so an unused decorated function does
// not break the Go build.
func emitDecoratedFunc(s *ast.FuncDef, ctx *GenContext) {
	// The generated wrapper keeps the original callable signature so call sites
	// (`greet(x)`) are unchanged. Inside, the decorators are applied to a
	// function literal of the body and the result is invoked with the params.
	//
	//	func greet(who any) any {
	//	    _dec := log(func(who any) any { <body> })
	//	    return _dec.(func(any) any)(who)
	//	}
	//
	// Decorators are `func(fn any) any` returning a `func(...any) any` closure;
	// the type assertion recovers a callable of the right arity.
	paramNames := make([]string, len(s.Params))
	params := make([]string, len(s.Params))
	anyList := make([]string, len(s.Params))
	for i, p := range s.Params {
		goT := "any"
		if p.Type != nil {
			goT = goType(p.Type, nil)
		}
		paramNames[i] = p.Name
		params[i] = p.Name + " " + goT
		anyList[i] = "any"
	}
	retT := "any"
	if s.RetType != nil {
		retT = goType(s.RetType, nil)
	}

	// Inner literal holding the undecorated body.
	var lit strings.Builder
	lit.WriteString(fmt.Sprintf("func(%s) %s {\n", strings.Join(paramsAsAny(s.Params), ", "), "any"))
	sub := &GenContext{PackageName: ctx.PackageName, Code: &lit, TempVarIdx: ctx.TempVarIdx, declared: map[string]bool{}, classes: ctx.classes}
	for _, st := range s.Body {
		EmitStmt(st, sub)
	}
	if !endsWithReturn(s.Body) {
		lit.WriteString("return nil\n")
	}
	lit.WriteString("}")
	ctx.TempVarIdx = sub.TempVarIdx

	// Apply decorators outermost (first) last so the nearest-to-def decorator
	// runs first. Each decorator is a value that takes the (single) function
	// being decorated and returns a wrapped function. A decorator written as a
	// bare name that is a known top-level function is called directly; anything
	// else (a decorator factory result like `retry(3)`, or a dynamic value) is
	// invoked through a `func(any) any` assertion so Go can call it.
	decorated := lit.String()
	for i := len(s.Decorators) - 1; i >= 0; i-- {
		dec := s.Decorators[i]
		callee := emitExpr(dec, ctx)
		if name, ok := dec.(*ast.Name); ok && ctx.isFunc(name.Ident) {
			decorated = fmt.Sprintf("%s(%s)", callee, decorated)
		} else {
			// Factory result or dynamic decorator: assert callability.
			decorated = fmt.Sprintf("%s.(func(any) any)(%s)", callee, decorated)
		}
	}

	// Callable type recovered from the decorator result via assertion.
	callableType := fmt.Sprintf("func(%s) any", strings.Join(anyList, ", "))

	ctx.markDeclared(s.Name)
	ctx.Code.WriteString(fmt.Sprintf("func %s(%s) %s {\n", s.Name, strings.Join(params, ", "), retT))
	ctx.Code.WriteString(fmt.Sprintf("_dec := %s\n", decorated))
	ret := fmt.Sprintf("_dec.(%s)(%s)", callableType, strings.Join(paramNames, ", "))
	if retT == "any" {
		ctx.Code.WriteString(fmt.Sprintf("return %s\n", ret))
	} else {
		// Assert the concrete return type when the def declares one.
		ctx.Code.WriteString(fmt.Sprintf("return %s.(%s)\n", ret, retT))
	}
	ctx.Code.WriteString("}\n")
}

// paramsAsAny renders a parameter list with every parameter typed `any`, used
// for the inner decorated literal whose signature must match the `func(...any)
// any` shape the decorator expects.
func paramsAsAny(ps []*ast.Param) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Name + " any"
	}
	return out
}

// goType maps a Rayo type annotation to its Go representation. Type-parameter

// goType maps a Rayo type annotation to its Go representation. Type-parameter
// names map to themselves; known basic types map to their Go equivalents;
// parameterized `list`/`dict` map to slices/maps; optionals map to pointers.
// Anything unknown falls back to `any`.
func goType(t ast.Type, typeParams map[string]bool) string {
	switch tt := t.(type) {
	case *ast.Any, nil:
		return "any"
	case *ast.Optional:
		return "*" + goType(tt.Elem, typeParams)
	case *ast.Named:
		if typeParams[tt.Name] {
			return tt.Name
		}
		switch tt.Name {
		case "int":
			return "int64"
		case "float":
			return "float64"
		case "str":
			return "string"
		case "bool":
			return "bool"
		case "any":
			return "any"
		case "list":
			if len(tt.Args) == 1 {
				return "[]" + goType(tt.Args[0], typeParams)
			}
			return "[]any"
		case "dict":
			if len(tt.Args) == 2 {
				return "map[" + goType(tt.Args[0], typeParams) + "]" + goType(tt.Args[1], typeParams)
			}
			return "map[string]any"
		default:
			// Unknown named type (e.g. a user class not yet modeled): stay
			// dynamic so codegen keeps compiling.
			return "any"
		}
	default:
		return "any"
	}
}

// zeroValue returns a Go zero value literal for a mapped Go type, used for the
// synthetic fallthrough return.
func zeroValue(goT string) string {
	switch goT {
	case "any":
		return "nil"
	case "int64":
		return "0"
	case "float64":
		return "0"
	case "string":
		return `""`
	case "bool":
		return "false"
	}
	if strings.HasPrefix(goT, "*") || strings.HasPrefix(goT, "[]") || strings.HasPrefix(goT, "map[") {
		return "nil"
	}
	// Type parameters and unknown types: use the Go zero-value idiom.
	return "*new(" + goT + ")"
}

// emitTry lowers try/except/finally to Go's recover-based error handling. The
// try body runs inside a closure whose panic is recovered; except handlers are
// matched in order and a finally block is emitted via defer.
func emitTry(s *ast.TryStmt, ctx *GenContext) {
	if len(s.Finally) > 0 {
		ctx.Code.WriteString("func() {\n")
		ctx.Code.WriteString("defer func() {\n")
		for _, st := range s.Finally {
			EmitStmt(st, ctx)
		}
		ctx.Code.WriteString("}()\n")
	}
	// Recover from raised errors and dispatch to except handlers.
	ctx.Code.WriteString("func() {\n")
	if len(s.Excepts) > 0 {
		errVar := "_exc"
		if s.Excepts[0].Var != "" {
			errVar = s.Excepts[0].Var
		}
		ctx.Code.WriteString("defer func() {\n")
		ctx.Code.WriteString(fmt.Sprintf("if %s := recover(); %s != nil {\n", errVar, errVar))
		ctx.Code.WriteString(fmt.Sprintf("_ = %s\n", errVar))
		for _, exc := range s.Excepts {
			for _, st := range exc.Body {
				EmitStmt(st, ctx)
			}
		}
		ctx.Code.WriteString("}\n")
		ctx.Code.WriteString("}()\n")
	}
	for _, st := range s.Body {
		EmitStmt(st, ctx)
	}
	ctx.Code.WriteString("}()\n")
	if len(s.Finally) > 0 {
		ctx.Code.WriteString("}()\n")
	}
}

// emitMatch lowers `match subject { case ... }` to a subject temp plus an
// if/else-if chain:
//   - literal/expression cases compare the subject for equality;
//   - capture cases bind the subject to a name and always match;
//   - the `_` wildcard becomes the trailing `else`.
//
// The subject is evaluated exactly once into a temp so side effects are not
// repeated, matching the semantics of a single-subject match.
func emitMatch(s *ast.MatchStmt, ctx *GenContext) {
	subj := ctx.NewTempVar()
	ctx.Code.WriteString(fmt.Sprintf("%s := %s\n", subj, emitExpr(s.Subject, ctx)))
	ctx.Code.WriteString(fmt.Sprintf("_ = %s\n", subj))

	// A capture/wildcard case always matches, so it is a "default" arm and any
	// cases after it are unreachable. Emit conditional (literal) arms as an
	// if/else-if chain; a single default arm becomes the trailing else (or a
	// bare block if it is the only arm).
	emittedIf := false
	for _, cs := range s.Cases {
		isDefault := cs.IsWildcard || cs.Binding != ""
		if isDefault {
			if emittedIf {
				ctx.Code.WriteString("} else {\n")
			} else {
				ctx.Code.WriteString("{\n")
			}
			if cs.Binding != "" {
				ctx.Code.WriteString(fmt.Sprintf("%s := %s\n", cs.Binding, subj))
				ctx.Code.WriteString(fmt.Sprintf("_ = %s\n", cs.Binding))
			}
			for _, st := range cs.Body {
				EmitStmt(st, ctx)
			}
			ctx.Code.WriteString("}\n")
			// Default is terminal; later cases are unreachable.
			return
		}

		// Literal/expression case: compare subject == pattern.
		cond := fmt.Sprintf("%s == %s", subj, emitExpr(cs.Pattern, ctx))
		if emittedIf {
			ctx.Code.WriteString(fmt.Sprintf("} else if %s {\n", cond))
		} else {
			ctx.Code.WriteString(fmt.Sprintf("if %s {\n", cond))
			emittedIf = true
		}
		for _, st := range cs.Body {
			EmitStmt(st, ctx)
		}
	}
	if emittedIf {
		ctx.Code.WriteString("}\n")
	}
}

// endsWithReturn reports whether the final statement of a body is a return, so
// the codegen can skip appending a synthetic `return nil`.
func endsWithReturn(stmts []ast.Stmt) bool {
	if len(stmts) == 0 {
		return false
	}
	_, ok := stmts[len(stmts)-1].(*ast.ReturnStmt)
	return ok
}

func emitExpr(expr ast.Expr, ctx *GenContext) string {
	switch e := expr.(type) {
	case *ast.Literal:
		return emitLiteral(e.Value)
	case *ast.Name:
		return e.Ident
	case *ast.UnaryOp:
		op := e.Op
		if op == "not" {
			op = "!"
		}
		return fmt.Sprintf("%s%s", op, emitExpr(e.Right, ctx))
	case *ast.BinaryOp:
		return fmt.Sprintf("(%s %s %s)", emitExpr(e.Left, ctx), goBinaryOp(e.Op), emitExpr(e.Right, ctx))
	case *ast.Call:
		return emitCall(e, ctx)
	case *ast.Index:
		return fmt.Sprintf("%s[%s]", emitExpr(e.Target, ctx), emitExpr(e.Index, ctx))
	case *ast.Attr:
		return fmt.Sprintf("%s.%s", emitExpr(e.Target, ctx), e.Attr)
	case *ast.DictLit:
		return LowerDict(e, ctx)
	case *ast.ListLit:
		return LowerList(e, ctx)
	case *ast.Lambda:
		return emitLambda(e, ctx)
	default:
		return "nil"
	}
}

// emitLiteral renders a literal value as valid Go source. Numbers are emitted
// bare; strings are Go-quoted; booleans and nil map directly.
func emitLiteral(v any) string {
	switch x := v.(type) {
	case string:
		return fmt.Sprintf("%q", x)
	case bool:
		return fmt.Sprintf("%v", x)
	case nil:
		return "nil"
	default: // int, int64, float64, etc.
		return fmt.Sprintf("%v", x)
	}
}

// goBinaryOp maps Rayo operators to their Go equivalents.
func goBinaryOp(op string) string {
	switch op {
	case "and":
		return "&&"
	case "or":
		return "||"
	default:
		return op
	}
}

func emitCall(e *ast.Call, ctx *GenContext) string {
	funcName := emitExpr(e.Func, ctx)
	if funcName == "print" {
		funcName = "fmt.Println"
	}
	args := make([]string, len(e.Args))
	for i, arg := range e.Args {
		args[i] = emitExpr(arg, ctx)
	}

	// A call whose callee is a declared class name constructs an instance;
	// rewrite `Point(...)` to the generated constructor `NewPoint(...)`.
	if name, ok := e.Func.(*ast.Name); ok && ctx.isClass(name.Ident) {
		return fmt.Sprintf("New%s(%s)", name.Ident, strings.Join(args, ", "))
	}

	// A call whose callee is a bare name that is NOT a known top-level function,
	// class, or the `print` builtin targets a dynamic `any` value (typically a
	// function passed as a parameter, e.g. inside a decorator). Go cannot call
	// an interface value directly, so assert it to a `func(...any) any` first.
	if name, ok := e.Func.(*ast.Name); ok && !ctx.isFunc(name.Ident) && name.Ident != "print" && name.Ident != "raise" {
		anyList := make([]string, len(e.Args))
		for i := range anyList {
			anyList[i] = "any"
		}
		return fmt.Sprintf("%s.(func(%s) any)(%s)", funcName, strings.Join(anyList, ", "), strings.Join(args, ", "))
	}

	return fmt.Sprintf("%s(%s)", funcName, strings.Join(args, ", "))
}

func emitLambda(e *ast.Lambda, ctx *GenContext) string {
	params := make([]string, len(e.Params))
	for i, p := range e.Params {
		params[i] = p.Name + " any"
	}
	return fmt.Sprintf("func(%s) any { return %s }", strings.Join(params, ", "), emitExpr(e.Body, ctx))
}
