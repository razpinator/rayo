package lex

import (
    "unicode"
    "strings"
)

// Python keywords (subset for demo; use full list in production)
var pythonKeywords = map[string]struct{}{
    "if": {}, "elif": {}, "else": {}, "while": {}, "for": {}, "def": {}, "return": {}, "try": {}, "except": {}, "finally": {}, "None": {},
}

// Lexer holds state for lexing.
type Lexer struct {
    src    string
    offset int
    line   int
    col    int
}

func NewLexer(src string) *Lexer {
    return &Lexer{src: src, line: 1, col: 1}
}

// Next returns the next token.
func (lx *Lexer) Next() Token {
    for lx.offset < len(lx.src) {
        ch := lx.src[lx.offset]
        switch ch {
        case ' ', '\t', '\r':
            start := lx.offset
            for lx.offset < len(lx.src) && (lx.src[lx.offset] == ' ' || lx.src[lx.offset] == '\t' || lx.src[lx.offset] == '\r') {
                lx.offset++
                lx.col++
            }
            return Token{Kind: TokenWhitespace, Value: lx.src[start:lx.offset], Offset: start, Line: lx.line, Col: lx.col}
        case '\n':
            lx.offset++
            lx.line++
            lx.col = 1
            return Token{Kind: TokenWhitespace, Value: "\n", Offset: lx.offset - 1, Line: lx.line - 1, Col: 1}
        case '#':
            start := lx.offset
            for lx.offset < len(lx.src) && lx.src[lx.offset] != '\n' {
                lx.offset++
                lx.col++
            }
            return Token{Kind: TokenComment, Value: lx.src[start:lx.offset], Offset: start, Line: lx.line, Col: lx.col}
        case '{':
            lx.offset++
            lx.col++
            return Token{Kind: TokenLBrace, Value: "{", Offset: lx.offset - 1, Line: lx.line, Col: lx.col - 1}
        case '}':
            lx.offset++
            lx.col++
            return Token{Kind: TokenRBrace, Value: "}", Offset: lx.offset - 1, Line: lx.line, Col: lx.col - 1}
        case '(':
            lx.offset++
            lx.col++
            return Token{Kind: TokenLParen, Value: "(", Offset: lx.offset - 1, Line: lx.line, Col: lx.col - 1}
        case ')':
            lx.offset++
            lx.col++
            return Token{Kind: TokenRParen, Value: ")", Offset: lx.offset - 1, Line: lx.line, Col: lx.col - 1}
        case '[':
            lx.offset++
            lx.col++
            return Token{Kind: TokenLBracket, Value: "[", Offset: lx.offset - 1, Line: lx.line, Col: lx.col - 1}
        case ']':
            lx.offset++
            lx.col++
            return Token{Kind: TokenRBracket, Value: "]", Offset: lx.offset - 1, Line: lx.line, Col: lx.col - 1}
        case ',':
            lx.offset++
            lx.col++
            return Token{Kind: TokenComma, Value: ",", Offset: lx.offset - 1, Line: lx.line, Col: lx.col - 1}
        case ':':
            lx.offset++
            lx.col++
            return Token{Kind: TokenColon, Value: ":", Offset: lx.offset - 1, Line: lx.line, Col: lx.col - 1}
        case '.':
            lx.offset++
            lx.col++
            return Token{Kind: TokenDot, Value: ".", Offset: lx.offset - 1, Line: lx.line, Col: lx.col - 1}
        case '"', '\'':
            quote := ch
            start := lx.offset
            lx.offset++
            lx.col++
            for lx.offset < len(lx.src) && lx.src[lx.offset] != quote {
                if lx.src[lx.offset] == '\n' {
                    lx.line++
                    lx.col = 1
                } else {
                    lx.col++
                }
                lx.offset++
            }
            if lx.offset < len(lx.src) {
                lx.offset++
                lx.col++
            }
            return Token{Kind: TokenString, Value: lx.src[start:lx.offset], Offset: start, Line: lx.line, Col: lx.col}
        default:
            if unicode.IsDigit(rune(ch)) {
                start := lx.offset
                for lx.offset < len(lx.src) && unicode.IsDigit(rune(lx.src[lx.offset])) {
                    lx.offset++
                    lx.col++
                }
                return Token{Kind: TokenNumber, Value: lx.src[start:lx.offset], Offset: start, Line: lx.line, Col: lx.col}
            }
            if unicode.IsLetter(rune(ch)) || ch == '_' {
                start := lx.offset
                for lx.offset < len(lx.src) && (unicode.IsLetter(rune(lx.src[lx.offset])) || unicode.IsDigit(rune(lx.src[lx.offset])) || lx.src[lx.offset] == '_') {
                    lx.offset++
                    lx.col++
                }
                val := lx.src[start:lx.offset]
                if _, ok := pythonKeywords[val]; ok {
                    return Token{Kind: TokenKeyword, Value: val, Offset: start, Line: lx.line, Col: lx.col}
                }
                return Token{Kind: TokenIdent, Value: val, Offset: start, Line: lx.line, Col: lx.col}
            }
            // Operators
            ops := "+-*/%==!=<><=>=&&||!"
            if strings.ContainsRune(ops, rune(ch)) {
                start := lx.offset
                lx.offset++
                lx.col++
                // Try to consume multi-char ops
                for lx.offset < len(lx.src) && strings.ContainsRune(ops, rune(lx.src[lx.offset])) {
                    lx.offset++
                    lx.col++
                }
                return Token{Kind: TokenOp, Value: lx.src[start:lx.offset], Offset: start, Line: lx.line, Col: lx.col}
            }
            // Unknown char: error token
            start := lx.offset
            lx.offset++
            lx.col++
            return Token{Kind: TokenError, Value: lx.src[start:lx.offset], Offset: start, Line: lx.line, Col: lx.col}
        }
    }
    return Token{Kind: TokenEOF, Value: "", Offset: lx.offset, Line: lx.line, Col: lx.col}
}
