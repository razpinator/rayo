package repl

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"rayo/internal/ast"
	"rayo/internal/compile"
	"rayo/internal/parse"
	"rayo/internal/sem"
)

// REPL evaluates Rayo in a persistent session (same pipeline as rayo run).
type REPL struct {
	opts       compile.Options
	sessionDir string
	accum      strings.Builder
	History    []string
}

func NewREPL(opts compile.Options) *REPL {
	dir, err := os.MkdirTemp("", "rayo-repl-*")
	if err != nil {
		dir = os.TempDir()
	}
	return &REPL{opts: opts, sessionDir: dir}
}

func (r *REPL) sessionFile() string {
	return filepath.Join(r.sessionDir, "session.ryo")
}

// Run starts the interactive loop until EOF or :quit.
func (r *REPL) Run() {
	defer func() { _ = os.RemoveAll(r.sessionDir) }()

	in := bufio.NewReader(os.Stdin)
	fmt.Println("Rayo REPL (same pipeline as rayo run). Type :help for commands.")
	for {
		chunk, err := r.readChunk(in)
		if err != nil {
			break
		}
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		r.History = append(r.History, chunk)

		if strings.HasPrefix(chunk, ":") {
			if r.handleMeta(chunk) {
				break
			}
			continue
		}

		if err := r.eval(chunk); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
	}
}

func (r *REPL) readChunk(in *bufio.Reader) (string, error) {
	var lines []string
	for {
		if len(lines) == 0 {
			fmt.Print("> ")
		} else {
			fmt.Print("... ")
		}
		line, err := in.ReadString('\n')
		if err != nil {
			if len(lines) == 0 {
				return "", err
			}
			break
		}
		line = strings.TrimRight(line, "\r\n")
		lines = append(lines, line)
		joined := strings.Join(lines, "\n")
		if !openBraces(joined) {
			break
		}
	}
	return strings.Join(lines, "\n"), nil
}

func openBraces(s string) bool {
	depth := 0
	for _, ch := range s {
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth > 0
}

func (r *REPL) handleMeta(line string) bool {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return false
	}
	switch parts[0] {
	case ":quit", ":q":
		return true
	case ":help":
		fmt.Println(":type <expr> — show inferred type")
		fmt.Println(":vars — list top-level names from the current session")
		fmt.Println(":quit — exit")
		return false
	case ":vars":
		r.printVars()
		return false
	case ":type":
		rest := strings.TrimSpace(strings.TrimPrefix(line, ":type"))
		if rest == "" {
			fmt.Fprintln(os.Stderr, "usage: :type <expression>")
			return false
		}
		expr, errs := parse.ParseExpr(rest)
		if len(errs) > 0 {
			fmt.Fprintf(os.Stderr, "parse: %v\n", errs)
			return false
		}
		fmt.Println(formatType(sem.InferType(expr)))
		return false
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q (try :help)\n", parts[0])
		return false
	}
}

func (r *REPL) printVars() {
	src := r.accum.String()
	if strings.TrimSpace(src) == "" {
		fmt.Println("(empty session)")
		return
	}
	parser := parse.NewParser(src)
	mod := parser.ParseModule()
	if len(parser.Errors()) > 0 {
		fmt.Fprintf(os.Stderr, "parse: %v\n", parser.Errors())
		return
	}
	seen := map[string]struct{}{}
	for _, stmt := range mod.Body {
		switch s := stmt.(type) {
		case *ast.VarStmt:
			if s.Name != "" {
				seen[s.Name] = struct{}{}
			}
		case *ast.FuncDef:
			if s.Name != "" {
				seen[s.Name] = struct{}{}
			}
		case *ast.AssignStmt:
			if n, ok := s.Target.(*ast.Name); ok {
				seen[n.Ident] = struct{}{}
			}
		}
	}
	if len(seen) == 0 {
		fmt.Println("(no bindings found)")
		return
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	fmt.Println("Names:")
	for _, n := range names {
		fmt.Printf("  %s\n", n)
	}
}

func (r *REPL) eval(chunk string) error {
	trial := r.accum.String()
	if trial != "" {
		trial += "\n"
	}
	trial += chunk

	if err := os.WriteFile(r.sessionFile(), []byte(trial), 0o644); err != nil {
		return err
	}
	goSrc, err := compile.BuildProgram(r.sessionFile(), r.opts)
	if err != nil {
		return err
	}

	if r.accum.Len() > 0 {
		r.accum.WriteByte('\n')
	}
	r.accum.WriteString(chunk)

	out, runErr := runGo(goSrc)
	if runErr != nil {
		return runErr
	}
	if out != "" {
		fmt.Print(out)
		if !strings.HasSuffix(out, "\n") {
			fmt.Println()
		}
	}
	return nil
}

func runGo(goSrc string) (string, error) {
	f, err := os.CreateTemp("", "rayo-repl-*.go")
	if err != nil {
		return "", err
	}
	path := f.Name()
	defer func() { _ = os.Remove(path) }()
	if _, err := f.WriteString(goSrc); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	var stdout, stderr strings.Builder
	cmd := exec.Command("go", "run", path)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w\n%s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func formatType(t sem.Type) string {
	switch x := t.(type) {
	case *sem.BasicType:
		return x.Name
	case *sem.OptionalType:
		return "optional(" + formatType(x.Elem) + ")"
	case *sem.AnyType:
		return "any"
	default:
		return fmt.Sprintf("%T", t)
	}
}
