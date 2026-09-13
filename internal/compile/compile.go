package compile

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"rayo/internal/ast"
	"rayo/internal/diag"
	"rayo/internal/gen"
	"rayo/internal/opt"
	"rayo/internal/parse"
	"rayo/internal/sem"
)

// Options configures multi-file compilation.
type Options struct {
	// IncludePaths are extra directories searched for imported .ryo modules.
	IncludePaths []string
}

// semReporter collects semantic diagnostics. It implements
// diag.SeverityReporter so warnings (e.g. unused variables) and info hints do
// not abort compilation; only errors are fatal.
type semReporter struct {
	errs  []string
	warns []string
}

func (s *semReporter) Report(span diag.Span, msg string) {
	s.ReportAt(span, diag.SeverityError, msg)
}

func (s *semReporter) ReportAt(_ diag.Span, sev diag.Severity, msg string) {
	if sev == diag.SeverityError {
		s.errs = append(s.errs, msg)
	} else {
		s.warns = append(s.warns, msg)
	}
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

// goImport is a collected Go import with an optional local alias (from
// `import "path" as name`). Rayo-source (.ryo) imports are inlined and never
// become Go imports, so they are filtered out before emission.
type goImport struct {
	Path  string
	Alias string
}

func collectModules(fromFile string, visit map[string]int, opts Options, stmts *[]ast.Stmt, imports *[]goImport) error {
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
	if len(rep.errs) > 0 {
		return fmt.Errorf("semantic errors in %s:\n  %s", fromFile, strings.Join(rep.errs, "\n  "))
	}

	// Optimize: constant folding, small-function inlining, and dead-code
	// elimination (behavior-preserving) before code generation.
	opt.Optimize(mod)

	for _, stmt := range mod.Body {
		if gen.ContainsPrint(stmt) {
			*imports = append(*imports, goImport{Path: "fmt"})
			break
		}
	}

	for _, imp := range mod.Imports {
		*imports = append(*imports, goImport{Path: imp.Path, Alias: imp.Alias})
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

func buildGoSource(stmts []ast.Stmt, imports []goImport) string {
	// Deduplicate Go imports by path, preserving the first alias seen. A .ryo
	// import is inlined elsewhere and never emitted as a Go import.
	aliasByPath := map[string]string{}
	var paths []string
	for _, imp := range imports {
		if imp.Path == "" || strings.HasSuffix(imp.Path, ".ryo") {
			continue
		}
		if _, seen := aliasByPath[imp.Path]; !seen {
			paths = append(paths, imp.Path)
		}
		if imp.Alias != "" {
			aliasByPath[imp.Path] = imp.Alias
		} else if _, seen := aliasByPath[imp.Path]; !seen {
			aliasByPath[imp.Path] = ""
		}
	}

	var pkg strings.Builder
	pkg.WriteString("package main\n\n")
	sort.Strings(paths)
	for _, path := range paths {
		if alias := aliasByPath[path]; alias != "" {
			pkg.WriteString(fmt.Sprintf("import %s %q\n", alias, path))
		} else {
			pkg.WriteString(fmt.Sprintf("import %q\n", path))
		}
	}

	var topFuncs strings.Builder
	var mainBody strings.Builder
	ctx := gen.NewGenContext("main")
	gen.RegisterClasses(stmts, ctx)

	for _, stmt := range stmts {
		if gen.IsTopLevelDecl(stmt) {
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

	return formatGo(pkg.String())
}

// formatGo runs the generated source through go/format so the output is
// gofmt-clean and go-vet friendly. If formatting fails (e.g. the generator
// produced syntactically invalid Go), the unformatted source is returned so
// the downstream `go run` surfaces a precise compiler error instead of an
// opaque format error.
func formatGo(src string) string {
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return src
	}
	return string(formatted)
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
	var imports []goImport
	if err := collectModules(mainPath, visit, opts, &stmts, &imports); err != nil {
		return "", err
	}
	return buildGoSource(stmts, imports), nil
}
