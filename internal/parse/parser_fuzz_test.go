//go:build go1.18

package parse

import (
	"os"
	"path/filepath"
	"testing"
)

// goldenSeeds returns the source of every testdata/golden/*.ryo fixture so the
// parser fuzzer starts from real programs. Missing corpus dirs are tolerated.
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

func FuzzParserRoundTrip(f *testing.F) {
	seeds := []string{
		"import 'core'\nvar x = 42",
		"if{x+1==2:foo}",
		"def foo(): return 42",
		"{a:1, b:2}",
		"print((1 + 2) - 3)",
		"def () {\n}\n",
		"config.name",
		"data[\"key\"]",
	}
	seeds = append(seeds, goldenSeeds()...)
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		p := NewParser(src)
		mod := p.ParseModule()
		if mod == nil {
			t.Fatal("ParseModule returned nil module")
		}
		// Any span the parser attached must be internally consistent.
		for _, stmt := range mod.Body {
			s := stmt.Span()
			if !s.IsZero() && s.End.Offset < s.Start.Offset {
				t.Fatalf("stmt span end before start: %+v", s)
			}
		}
	})
}
