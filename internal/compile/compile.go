package compile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"rayo/internal/ast"
	"rayo/internal/diag"
	"rayo/internal/gen"
	"rayo/internal/parse"
	"rayo/internal/sem"
)

// Options configures multi-file compilation.
type Options struct {
	// IncludePaths are extra directories searched for imported .ryo modules.
	IncludePaths []string
}

type semReporter struct {
	msgs []string
}

func (s *semReporter) Report(_ diag.Span, msg string) {
	s.msgs = append(s.msgs, msg)
}

func absKey(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty module path")
	}
	return filepath.Abs(filepath.Clean(path))
}

// resolveRyoImport resolves impPath from the importing file's directory and include paths.
func resolveRyoImport(fromFile, impPath string, includePaths []string) (string, error) {
	if !strings.HasSuffix(impPath, ".ryo") {
		return "", fmt.Errorf("internal error: resolveRyoImport on non-.ryo path %q", impPath)
	}
	dir := filepath.Dir(fromFile)
	candidates := []string{}
	if strings.HasPrefix(impPath, "./") || strings.HasPrefix(impPath, "../") {
		candidates = append(candidates, filepath.Clean(filepath.Join(dir, impPath)))
	} else {
		candidates = append(candidates, filepath.Clean(filepath.Join(dir, impPath)))
	}
	for _, inc := range includePaths {
		candidates = append(candidates, filepath.Clean(filepath.Join(inc, impPath)))
	}
	var tried []string
	for _, c := range candidates {
		tried = append(tried, c)
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("cannot resolve import %q from %s (tried: %s)",
		impPath, fromFile, strings.Join(tried, ", "))
}

const (
	visitNone = iota
	visitOpen
	visitDone
)

func collectModules(fromFile string, visit map[string]int, opts Options, stmts *[]ast.Stmt, imports *[]string) error {
	key, err := absKey(fromFile)
	if err != nil {
		return fmt.Errorf("%s: %w", fromFile, err)
	}
	switch visit[key] {
	case visitOpen:
		return fmt.Errorf("import cycle detected (re-entered %s)", key)
	case visitDone:
		return nil
	}
	visit[key] = visitOpen
	defer func() { visit[key] = visitDone }()

	src, err := os.ReadFile(fromFile)
	if err != nil {
		return fmt.Errorf("reading %s: %w", fromFile, err)
	}

	parser := parse.NewParser(string(src))
	mod := parser.ParseModule()
	if len(parser.Errors()) > 0 {
		return fmt.Errorf("parse errors in %s: %v", fromFile, parser.Errors())
	}

	rep := &semReporter{}
	sem.CheckModule(mod, rep)
	if len(rep.msgs) > 0 {
		return fmt.Errorf("semantic issues in %s:\n  %s", fromFile, strings.Join(rep.msgs, "\n  "))
	}

	for _, stmt := range mod.Body {
		if gen.ContainsPrint(stmt) {
			*imports = append(*imports, "fmt")
			break
		}
	}

	for _, imp := range mod.Imports {
		*imports = append(*imports, imp.Path)
		if strings.HasSuffix(imp.Path, ".ryo") {
			resolved, err := resolveRyoImport(fromFile, imp.Path, opts.IncludePaths)
			if err != nil {
				return fmt.Errorf("%s: %w", fromFile, err)
			}
			if err := collectModules(resolved, visit, opts, stmts, imports); err != nil {
				return err
			}
		}
	}

	*stmts = append(*stmts, mod.Body...)
	return nil
}

func buildGoSource(stmts []ast.Stmt, imports []string) string {
	importSet := map[string]bool{}
	for _, imp := range imports {
		if imp == "" || strings.HasSuffix(imp, ".ryo") {
			continue
		}
		importSet[imp] = true
	}

	var pkg strings.Builder
	pkg.WriteString("package main\n\n")
	impKeys := make([]string, 0, len(importSet))
	for imp := range importSet {
		impKeys = append(impKeys, imp)
	}
	sort.Strings(impKeys)
	for _, imp := range impKeys {
		pkg.WriteString(fmt.Sprintf("import %q\n", imp))
	}

	var topFuncs strings.Builder
	var mainBody strings.Builder
	ctx := gen.NewGenContext("main")

	for _, stmt := range stmts {
		if _, ok := stmt.(*ast.FuncDef); ok {
			ctx.Code = &topFuncs
			gen.EmitStmt(stmt, ctx)
		} else {
			ctx.Code = &mainBody
			gen.EmitStmt(stmt, ctx)
		}
	}

	pkg.WriteString(topFuncs.String())
	if mainBody.Len() > 0 {
		pkg.WriteString("func main() {\n")
		pkg.WriteString(mainBody.String())
		pkg.WriteString("}\n")
	}

	return pkg.String()
}

// BuildProgram parses Rayo modules starting at mainPath (including .ryo imports),
// runs semantic checks, and returns a single Go source file.
func BuildProgram(mainPath string, opts Options) (string, error) {
	mainPath, err := filepath.Abs(filepath.Clean(mainPath))
	if err != nil {
		return "", err
	}
	visit := map[string]int{}
	var stmts []ast.Stmt
	var imports []string
	if err := collectModules(mainPath, visit, opts, &stmts, &imports); err != nil {
		return "", err
	}
	return buildGoSource(stmts, imports), nil
}
