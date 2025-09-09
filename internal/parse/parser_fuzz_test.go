//go:build go1.18

package parse

import "testing"

func FuzzParserRoundTrip(f *testing.F) {
    seeds := []string{
        "import 'core'\nvar x = 42",
        "if{x+1==2:foo}",
        "def foo(): return 42",
        "{a:1, b:2}",
    }
    for _, s := range seeds {
        f.Add(s)
    }
    f.Fuzz(func(t *testing.T, src string) {
        p := NewParser(src)
        _ = p.ParseModule()
    })
}
