package fmt

import (
    "testing"
    "functure/internal/lex"
)

func TestFormatTokens_Idempotent(t *testing.T) {
    src := "if{x+1==2:foo}" // intentionally unformatted
    lx := lex.NewLexer(src)
    var tokens []lex.Token
    for {
        tok := lx.Next()
        if tok.Kind == lex.TokenEOF {
            break
        }
        tokens = append(tokens, tok)
    }
    out := FormatTokens(tokens)
    lx2 := lex.NewLexer(out)
    var tokens2 []lex.Token
    for {
        tok := lx2.Next()
        if tok.Kind == lex.TokenEOF {
            break
        }
        tokens2 = append(tokens2, tok)
    }
    out2 := FormatTokens(tokens2)
    if out != out2 {
        t.Errorf("Formatter not idempotent: %q vs %q", out, out2)
    }
}
