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

// TestLexer_Positions verifies that every token reports its START line/col and
// byte offset. This guards the multi-char column-tracking fix (identifiers,
// numbers, strings, and operators previously reported their END column).
func TestLexer_Positions(t *testing.T) {
	type want struct {
		kind   TokenKind
		value  string
		offset int
		line   int
		col    int
	}
	cases := []struct {
		name string
		src  string
		want []want
	}{
		{
			name: "single line idents and op",
			src:  "foo + barbaz",
			want: []want{
				{TokenIdent, "foo", 0, 1, 1},
				{TokenWhitespace, " ", 3, 1, 4},
				{TokenOp, "+", 4, 1, 5},
				{TokenWhitespace, " ", 5, 1, 6},
				{TokenIdent, "barbaz", 6, 1, 7},
			},
		},
		{
			name: "number and string start columns",
			src:  "12345 'hello'",
			want: []want{
				{TokenNumber, "12345", 0, 1, 1},
				{TokenWhitespace, " ", 5, 1, 6},
				{TokenString, "'hello'", 6, 1, 7},
			},
		},
		{
			name: "multi-line ident position",
			src:  "a\n  bcd",
			want: []want{
				{TokenIdent, "a", 0, 1, 1},
				{TokenWhitespace, "\n", 1, 1, 1},
				{TokenWhitespace, "  ", 2, 2, 1},
				{TokenIdent, "bcd", 4, 2, 3},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lx := NewLexer(tc.src)
			for i, w := range tc.want {
				tok := lx.Next()
				if tok.Kind != w.kind || tok.Value != w.value || tok.Offset != w.offset || tok.Line != w.line || tok.Col != w.col {
					t.Errorf("token %d: got {kind:%v value:%q off:%d line:%d col:%d}, want {kind:%v value:%q off:%d line:%d col:%d}",
						i, tok.Kind, tok.Value, tok.Offset, tok.Line, tok.Col,
						w.kind, w.value, w.offset, w.line, w.col)
				}
			}
		})
	}
}
