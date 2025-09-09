//go:build go1.18

package lex

import "testing"

func FuzzLexerRoundTrip(f *testing.F) {
    seeds := []string{
        "if{x+1==2:foo}",
        "def foo(): return 42",
        "{a:1, b:2}",
        "try{...}except{...}finally{...}",
    }
    for _, s := range seeds {
        f.Add(s)
    }
    f.Fuzz(func(t *testing.T, src string) {
        lx := NewLexer(src)
        for {
            tok := lx.Next()
            if tok.Kind == TokenEOF {
                break
            }
        }
    })
}
