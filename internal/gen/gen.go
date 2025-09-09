package gen

import (
    "functure/internal/ast"
    "strings"
)

// GenContext holds state for code generation.
type GenContext struct {
    PackageName string
    Imports     []string
    TempVarIdx  int
    Code        *strings.Builder
}

func NewGenContext(pkg string) *GenContext {
    return &GenContext{PackageName: pkg, Code: &strings.Builder{}}
}

func (ctx *GenContext) NewTempVar() string {
    ctx.TempVarIdx++
    return "_tmp" + string(rune(ctx.TempVarIdx+48))
}
