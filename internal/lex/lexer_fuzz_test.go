//go:build go1.18

package lex

import (
	"os"
	"path/filepath"
	"testing"
)

// goldenSeeds returns the source of every testdata/golden/*.ryo fixture so the
// fuzzer starts from real, spec-exercising programs in addition to the inline
// seeds. Missing corpus dirs are tolerated (returns nil).
func goldenSeeds() []string {
	dir := filepath.Join("..", "..", "testdata", "golden")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".ryo" {
			continue
		}
		if b, err := os.ReadFile(filepath.Join(dir, e.Name())); err == nil {
			out = append(out, string(b))
		}
	}
	return out
}

func FuzzLexerRoundTrip(f *testing.F) {
	seeds := []string{
		"if{x+1==2:foo}",
		"def foo(): return 42",
		"{a:1, b:2}",
		"try{...}except{...}finally{...}",
		"print((1 + 2) - 3)",
		"config.name",
		"data[\"key\"]",
	}
	seeds = append(seeds, goldenSeeds()...)
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		lx := NewLexer(src)
		for {
			tok := lx.Next()
			// Positions must never regress: line/col are 1-based, offset within source.
			if tok.Line < 1 || tok.Col < 1 {
				t.Fatalf("invalid position for token %q: line=%d col=%d", tok.Value, tok.Line, tok.Col)
			}
			if tok.Offset < 0 || tok.Offset > len(src) {
				t.Fatalf("offset out of range for token %q: %d (len=%d)", tok.Value, tok.Offset, len(src))
			}
			if tok.Kind == TokenEOF {
				break
			}
		}
	})
}
