package lex

// TokenKind enumerates all token types.
type TokenKind int

const (
    TokenEOF TokenKind = iota
    TokenIdent
    TokenNumber
    TokenString
    TokenLBrace // {
    TokenRBrace // }
    TokenLParen // (
    TokenRParen // )
    TokenLBracket // [
    TokenRBracket // ]
    TokenComma // ,
    TokenColon // :
    TokenDot // .
    TokenOp // + - * / % == != < > <= >= && || !
    TokenKeyword
    TokenComment
    TokenWhitespace
    TokenError
)

// Token represents a single token.
type Token struct {
    Kind   TokenKind
    Value  string
    Offset int // byte offset
    Line   int // 1-based line
    Col    int // 1-based col
}
