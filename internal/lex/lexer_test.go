package lex

import (
    "testing"
)

func TestLexer_TableDriven(t *testing.T) {
    cases := []struct {
        name string
        src  string
        want []TokenKind
    }{
        {"identifiers", "foo bar", []TokenKind{TokenIdent, TokenWhitespace, TokenIdent, TokenEOF}},
        {"keywords", "if elif else", []TokenKind{TokenKeyword, TokenWhitespace, TokenKeyword, TokenWhitespace, TokenKeyword, TokenEOF}},
        {"numbers", "123 456", []TokenKind{TokenNumber, TokenWhitespace, TokenNumber, TokenEOF}},
        {"braces", "{ }", []TokenKind{TokenLBrace, TokenWhitespace, TokenRBrace, TokenEOF}},
        {"string", "'abc' \"def\"", []TokenKind{TokenString, TokenWhitespace, TokenString, TokenEOF}},
        {"comment", "# hello\nfoo", []TokenKind{TokenComment, TokenWhitespace, TokenIdent, TokenEOF}},
        {"ops", "+ - == !=", []TokenKind{TokenOp, TokenWhitespace, TokenOp, TokenWhitespace, TokenOp, TokenWhitespace, TokenOp, TokenEOF}},
        {"error", "@", []TokenKind{TokenError, TokenEOF}},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            lx := NewLexer(tc.src)
            var got []TokenKind
            for {
                tok := lx.Next()
                got = append(got, tok.Kind)
                if tok.Kind == TokenEOF {
                    break
                }
            }
            if len(got) != len(tc.want) {
                t.Errorf("case %q: got %v, want %v", tc.name, got, tc.want)
                return
            }
            for i := range got {
                if got[i] != tc.want[i] {
                    t.Errorf("case %q: token %d got %v, want %v", tc.name, i, got[i], tc.want[i])
                }
            }
        })
    }
}
