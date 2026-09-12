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
}

func NewGenContext(pkg string) *GenContext {
	return &GenContext{PackageName: pkg, Code: &strings.Builder{}, declared: map[string]bool{}}
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
