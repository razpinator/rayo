package golden

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"rayo/internal/ast"
	"rayo/internal/compile"
	"rayo/internal/diag"
	"rayo/internal/lex"
	"rayo/internal/parse"
	"rayo/internal/sem"
)

// GoldenCase is one testdata/golden/<name>.ryo fixture and optional sidecars.
type GoldenCase struct {
	Name   string
	Source string
	Expect map[string]string
}

// LoadGoldenCases loads .ryo sources and optional .out, .tokens, .ast, .go expectations.
func LoadGoldenCases(dir string) ([]GoldenCase, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var cases []GoldenCase
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ryo") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".ryo")
		srcPath := filepath.Join(dir, e.Name())
		srcBytes, err := os.ReadFile(srcPath)
		if err != nil {
			return nil, err
		}
		expect := map[string]string{}
		for _, ext := range []string{".out", ".tokens", ".ast", ".go", ".diags"} {
			outPath := filepath.Join(dir, base+ext)
			if st, err := os.Stat(outPath); err == nil && !st.IsDir() {
				b, _ := os.ReadFile(outPath)
				expect[ext[1:]] = string(b)
			}
		}
		cases = append(cases, GoldenCase{
			Name:   base,
			Source: string(srcBytes),
			Expect: expect,
		})
	}
	return cases, nil
}

// Diff returns a simple line diff; empty if equal.
func Diff(expected, actual string) string {
	if expected == actual {
		return ""
	}
	expLines := strings.Split(expected, "\n")
	actLines := strings.Split(actual, "\n")
	var buf bytes.Buffer
	max := len(expLines)
	if len(actLines) > max {
		max = len(actLines)
	}
	for i := 0; i < max; i++ {
		var exp, act string
		if i < len(expLines) {
			exp = expLines[i]
		}
		if i < len(actLines) {
			act = actLines[i]
		}
		if exp != act {
			fmt.Fprintf(&buf, "- %s\n+ %s\n", exp, act)
		}
	}
	return buf.String()
}

// diagDump renders parser and semantic diagnostics for a source in a stable,
// snapshot-friendly format: one diagnostic per line as
// "<line>:<col>: <severity>: <message>", sorted by position then message.
func diagDump(src string, mod *ast.Module, parser *parse.Parser) string {
	type entry struct {
		line, col int
		sev, msg  string
	}
	var entries []entry

	for _, err := range parser.Errors() {
		if pe, ok := err.(*parse.ParseError); ok {
			entries = append(entries, entry{pe.Span.Start.Line, pe.Span.Start.Col, "error", pe.Msg})
		} else {
			entries = append(entries, entry{0, 0, "error", err.Error()})
		}
	}

	rep := &diagCollector{}
	sem.CheckModule(mod, rep)
	for _, d := range rep.diags {
		entries = append(entries, entry{d.span.Start.Line, d.span.Start.Col, d.sev.String(), d.msg})
	}

	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.line != b.line {
			return a.line < b.line
		}
		if a.col != b.col {
			return a.col < b.col
		}
		return a.msg < b.msg
	})

	var sb strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&sb, "%d:%d: %s: %s\n", e.line, e.col, e.sev, e.msg)
	}
	return sb.String()
}

// diagCollector captures semantic diagnostics with their severity.
type diagCollector struct {
	diags []struct {
		span diag.Span
		sev  diag.Severity
		msg  string
	}
}

func (c *diagCollector) Report(span diag.Span, msg string) {
	c.ReportAt(span, diag.SeverityError, msg)
}

func (c *diagCollector) ReportAt(span diag.Span, sev diag.Severity, msg string) {
	c.diags = append(c.diags, struct {
		span diag.Span
		sev  diag.Severity
		msg  string
	}{span, sev, msg})
}

func tokenDump(src string) string {
	lx := lex.NewLexer(src)
	var b strings.Builder
	for {
		t := lx.Next()
		if t.Kind == lex.TokenEOF {
			break
		}
		if t.Kind == lex.TokenWhitespace || t.Kind == lex.TokenComment {
			continue
		}
		fmt.Fprintf(&b, "%s %q\n", tokenKindName(t.Kind), t.Value)
	}
	return b.String()
}

func tokenKindName(k lex.TokenKind) string {
	switch k {
	case lex.TokenIdent:
		return "Ident"
	case lex.TokenNumber:
		return "Number"
	case lex.TokenString:
		return "String"
	case lex.TokenLBrace:
		return "LBrace"
	case lex.TokenRBrace:
		return "RBrace"
	case lex.TokenLParen:
		return "LParen"
	case lex.TokenRParen:
		return "RParen"
	case lex.TokenLBracket:
		return "LBracket"
	case lex.TokenRBracket:
		return "RBracket"
	case lex.TokenComma:
		return "Comma"
	case lex.TokenColon:
		return "Colon"
	case lex.TokenDot:
		return "Dot"
	case lex.TokenOp:
		return "Op"
	case lex.TokenKeyword:
		return "Keyword"
	case lex.TokenError:
		return "Error"
	default:
		return fmt.Sprintf("Token(%d)", int(k))
	}
}

// Run executes golden checks under dir. caseFilter matches substrings of case names (empty = all).
func Run(dir, caseFilter string) error {
	cases, err := LoadGoldenCases(dir)
	if err != nil {
		return err
	}
	if len(cases) == 0 {
		return fmt.Errorf("no golden cases in %s", dir)
	}
	var errs []error
	matched := 0
	for _, c := range cases {
		if caseFilter != "" && !strings.Contains(c.Name, caseFilter) {
			continue
		}
		matched++
		if err := runOne(dir, c); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", c.Name, err))
		}
	}
	if caseFilter != "" && matched == 0 {
		return fmt.Errorf("no golden case matched filter %q", caseFilter)
	}
	return errors.Join(errs...)
}

func runOne(dir string, c GoldenCase) error {
	parser := parse.NewParser(c.Source)
	mod := parser.ParseModule()

	// Diagnostic-snapshot cases (bad programs). When a .diags sidecar exists we
	// expect diagnostics rather than a clean compile, so tolerate parse errors
	// and compare the collected diagnostics against the snapshot.
	if want, ok := c.Expect["diags"]; ok {
		got := diagDump(c.Source, mod, parser)
		if d := Diff(strings.TrimRight(want, "\n"), strings.TrimRight(got, "\n")); d != "" {
			return fmt.Errorf("diags mismatch:\n%s", d)
		}
		// Still allow other pure-syntax expectations (tokens/ast) below, but
		// skip compile/run which are meaningless for a diagnostic case.
		if _, hasTokens := c.Expect["tokens"]; hasTokens {
			if d := Diff(c.Expect["tokens"], tokenDump(c.Source)); d != "" {
				return fmt.Errorf("tokens mismatch:\n%s", d)
			}
		}
		return nil
	}

	if len(parser.Errors()) > 0 {
		return fmt.Errorf("parse: %v", parser.Errors())
	}

	ryoPath := filepath.Join(dir, c.Name+".ryo")
	opts := compile.Options{}

	if want, ok := c.Expect["tokens"]; ok {
		got := tokenDump(c.Source)
		if d := Diff(want, got); d != "" {
			return fmt.Errorf("tokens mismatch:\n%s", d)
		}
	}
	if want, ok := c.Expect["ast"]; ok {
		got := parse.PrettyPrint(mod)
		if d := Diff(want, got); d != "" {
			return fmt.Errorf("ast mismatch:\n%s", d)
		}
	}

	if want, ok := c.Expect["go"]; ok {
		got, err := compile.BuildProgram(ryoPath, opts)
		if err != nil {
			return fmt.Errorf("compile (.go check): %w", err)
		}
		if d := Diff(want, got); d != "" {
			return fmt.Errorf("go mismatch:\n%s", d)
		}
	}

	if want, ok := c.Expect["out"]; ok {
		goSrc, err := compile.BuildProgram(ryoPath, opts)
		if err != nil {
			return fmt.Errorf("compile (.out check): %w", err)
		}
		got, err := runGoSnippet(goSrc)
		if err != nil {
			return fmt.Errorf("run: %w", err)
		}
		if strings.TrimSpace(want) != strings.TrimSpace(got) {
			return fmt.Errorf("out mismatch:\n%s", Diff(want, got))
		}
	}

	if len(c.Expect) == 0 {
		if _, err := compile.BuildProgram(ryoPath, opts); err != nil {
			return fmt.Errorf("smoke compile: %w", err)
		}
	}
	return nil
}

func runGoSnippet(goSrc string) (string, error) {
	f, err := os.CreateTemp("", "rayo-golden-*.go")
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
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", path)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
