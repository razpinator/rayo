package gen

import (
	"fmt"
	"strings"
)

// GenContext holds state for code generation.
type GenContext struct {
	PackageName string
	Imports     []string
	TempVarIdx  int
	Code        *strings.Builder
	// declared tracks plain-name targets already introduced in the current
	// emission unit so the first assignment declares (`:=`) and later ones
	// reassign (`=`), avoiding Go's "no new variables on left side" error.
	declared map[string]bool
	// classes records declared class names so a call whose callee is a class
	// name is rewritten to the generated `New<Class>` constructor.
	classes map[string]bool
	// funcs records top-level function names emitted as real Go funcs. A call
	// whose callee is NOT one of these (and not a class or builtin) targets a
	// dynamic `any` value (e.g. a function passed as a parameter), so it is
	// invoked via a `func(...any) any` type assertion.
	funcs map[string]bool
}

func NewGenContext(pkg string) *GenContext {
	return &GenContext{PackageName: pkg, Code: &strings.Builder{}, declared: map[string]bool{}, classes: map[string]bool{}, funcs: map[string]bool{}}
}

// registerFunc records name as a top-level function.
func (ctx *GenContext) registerFunc(name string) {
	if ctx.funcs == nil {
		ctx.funcs = map[string]bool{}
	}
	ctx.funcs[name] = true
}

// isFunc reports whether name is a known top-level function.
func (ctx *GenContext) isFunc(name string) bool {
	return ctx.funcs != nil && ctx.funcs[name]
}

// registerClass records a class name so constructor calls can be rewritten.
func (ctx *GenContext) registerClass(name string) {
	if ctx.classes == nil {
		ctx.classes = map[string]bool{}
	}
	ctx.classes[name] = true
}

// isClass reports whether name is a declared class in this emission unit.
func (ctx *GenContext) isClass(name string) bool {
	return ctx.classes != nil && ctx.classes[name]
}

// markDeclared records name as declared and reports whether this is the first
// time it has been seen (i.e. it should use `:=`).
func (ctx *GenContext) markDeclared(name string) (first bool) {
	if ctx.declared == nil {
		ctx.declared = map[string]bool{}
	}
	if ctx.declared[name] {
		return false
	}
	ctx.declared[name] = true
	return true
}

// NewTempVar returns a fresh, collision-free temporary identifier. The previous
// implementation used rune arithmetic (`_tmp` + rune(idx+48)) which produced
// invalid identifiers past index 9; this uses decimal formatting instead.
func (ctx *GenContext) NewTempVar() string {
	ctx.TempVarIdx++
	return fmt.Sprintf("_tmp%d", ctx.TempVarIdx)
}
