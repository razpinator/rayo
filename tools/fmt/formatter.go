package fmt

import (
	"rayo/internal/lex"
	"strings"
)

// FormatTokens formats a token stream with stable rules.
func FormatTokens(tokens []lex.Token) string {
	var sb strings.Builder
	for i, tok := range tokens {
		switch tok.Kind {
		case lex.TokenWhitespace, lex.TokenComment, lex.TokenEOF:
			// Skip whitespace, comments, and EOF
			continue
		case lex.TokenLBrace, lex.TokenRBrace:
			sb.WriteString(tok.Value)
			sb.WriteString(" ")
		case lex.TokenOp:
			sb.WriteString(" ")
			sb.WriteString(tok.Value)
			sb.WriteString(" ")
		case lex.TokenComma:
			sb.WriteString(", ")
		case lex.TokenColon:
			sb.WriteString(": ")
		case lex.TokenKeyword:
			if i > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(tok.Value)
			sb.WriteString(" ")
		default:
			sb.WriteString(tok.Value)
			sb.WriteString(" ")
		}
	}
	return strings.TrimSpace(sb.String())
}
