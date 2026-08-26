package golden

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"rayo/internal/compile"
	"rayo/internal/lex"
	"rayo/internal/parse"
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
		for _, ext := range []string{".out", ".tokens", ".ast", ".go"} {
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
